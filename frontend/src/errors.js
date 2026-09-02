// friendlyError traduit une erreur d'appel API en message clair en français.
// Utilisé partout pour éviter d'exposer des messages techniques bruts.
export function friendlyError(e, fallback = 'Une erreur est survenue.') {
  if (!e) return fallback
  switch (e.status) {
    case 423:
      return 'Coffre verrouillé — déverrouillez-le pour cette action.'
    case 400:
      return (e.data && e.data.error) || 'Requête invalide (vérifiez les champs).'
    case 401:
      return 'Session expirée — rechargez la page (Ctrl+Shift+R).'
    case 403:
      return (e.data && e.data.error) || 'Action refusée.'
    case 404:
      return 'Ressource introuvable.'
    case 502:
    case 504:
      return 'Équipement injoignable ou délai dépassé.'
  }
  // Le backend renvoie généralement des messages déjà en français.
  return e.message || fallback
}
