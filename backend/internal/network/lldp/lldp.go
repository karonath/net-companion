// Package lldp parse les trames LLDP et expose une capture à dégradation propre.
package lldp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// Neighbor est un voisin LLDP (switch/port en face).
type Neighbor struct {
	ChassisID       string `json:"chassisId"`
	PortID          string `json:"portId"`
	SystemName      string `json:"systemName"`
	PortDescription string `json:"portDescription"`
}

// Result est le retour d'une capture LLDP.
type Result struct {
	Available bool       `json:"available"`
	Reason    string     `json:"reason,omitempty"`
	Neighbors []Neighbor `json:"neighbors"`
}

const ethertypeLLDP = 0x88CC

// ParseEthernet valide l'ethertype LLDP puis parse les TLV.
func ParseEthernet(frame []byte) (Neighbor, error) {
	if len(frame) < 14 {
		return Neighbor{}, errors.New("trame trop courte")
	}
	if binary.BigEndian.Uint16(frame[12:14]) != ethertypeLLDP {
		return Neighbor{}, errors.New("ethertype non-LLDP")
	}
	return parseTLVs(frame[14:])
}

func parseTLVs(b []byte) (Neighbor, error) {
	var n Neighbor
	i := 0
	for i+2 <= len(b) {
		hdr := binary.BigEndian.Uint16(b[i : i+2])
		typ := int(hdr >> 9)
		length := int(hdr & 0x1FF)
		i += 2
		if typ == 0 { // End of LLDPDU
			break
		}
		if i+length > len(b) {
			return n, errors.New("TLV tronqué")
		}
		val := b[i : i+length]
		switch typ {
		case 1:
			n.ChassisID = decodeID(val)
		case 2:
			n.PortID = decodeID(val)
		case 4:
			n.PortDescription = string(val)
		case 5:
			n.SystemName = string(val)
		}
		i += length
	}
	return n, nil
}

// decodeID interprète un Chassis/Port ID : 1er octet = subtype, reste = valeur.
// Un identifiant de 6 octets est rendu comme une MAC, sinon comme une chaîne.
func decodeID(val []byte) string {
	if len(val) < 1 {
		return ""
	}
	body := val[1:]
	if len(body) == 6 {
		return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			body[0], body[1], body[2], body[3], body[4], body[5])
	}
	return string(body)
}

// captureTimeout est la durée d'écoute par défaut (utilisée par l'impl npcap).
const captureTimeout = 6 * time.Second
