package api

// Referred to by the UmoIQ api as "direction", this is a route's 
// service variant describing the order of stops the service uses, 
// eg for a bus there may be North/South or East/West routes. These
// routes may not use the same stops, or they might. Stops may be
// accessed on a per-service basis, or if you know that the route's 
// services share stops, you may call `Route.GetStops()` to retrieve
// a list of all stops used by any service.
type Service struct {
  Tag           string
  Name          string
  Title         string
  UseForUI      bool
  Stops         []*Stop
  api           *ApiHandler
  agency        *Agency
  route         *Route
}
