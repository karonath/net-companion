// Package cloud remonte les snapshots d'intervention vers un endpoint
// d'ingestion HTTPS, de façon optionnelle et best-effort. Si le cloud n'est pas
// configuré, le paquet est un no-op : l'agent fonctionne intégralement en local.
package cloud

import "os"

// Config est lue depuis l'environnement du technicien (jamais depuis le binaire).
type Config struct {
	URL      string // NC_CLOUD_URL : endpoint d'ingestion
	Key      string // NC_CLOUD_KEY : clé d'API du technicien
	ClientID string // NC_CLIENT_ID : client de l'intervention
	SiteID   string // NC_SITE_ID   : site de l'intervention
	TechID   string // NC_TECH_ID   : identifiant technicien (optionnel)
}

// ConfigFromEnv lit la configuration cloud depuis les variables d'environnement.
func ConfigFromEnv() Config {
	return Config{
		URL:      os.Getenv("NC_CLOUD_URL"),
		Key:      os.Getenv("NC_CLOUD_KEY"),
		ClientID: os.Getenv("NC_CLIENT_ID"),
		SiteID:   os.Getenv("NC_SITE_ID"),
		TechID:   os.Getenv("NC_TECH_ID"),
	}
}

// Enabled indique si la remontée cloud est activée (endpoint + clé présents).
func (c Config) Enabled() bool { return c.URL != "" && c.Key != "" }

// Complete indique si le rattachement client/site est aussi renseigné.
func (c Config) Complete() bool { return c.Enabled() && c.ClientID != "" && c.SiteID != "" }
