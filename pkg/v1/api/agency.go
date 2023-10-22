package api

import (
	"encoding/json"
  "slices"
	"errors"
	"time"

	"github.com/lcyvin/go-umoparse/internal/utils"
)

type Agency struct {
  Title       string
  Tag         string
  ShortTitle  string
  RegionTitle string
  Routes      []*Route
  api         *ApiHandler
  cacheAge    time.Time
}

func GetAgency(agencyTag string, opts...ApiHandlerOption) (*Agency, error) {
  return DefaultApiHandler.GetAgency(agencyTag, opts...)
}

func (a *Agency) GetRoute(routeTag string) (*Route, error) {
  if a.Routes == nil {
    _, err := a.GetRoutes()
    if err != nil {
      return nil, err
    }
  }

  for _, route := range a.Routes {
    if route.Tag == routeTag {
      return route, nil
    }
  }

  return nil, errors.New("RouteNotFoundErr")
}

func (a *Agency) GetService(svcTag string) (*Service, error) {
  routes, err := a.GetRoutes()
  if err != nil {
    return nil, err
  }

  for _, route := range routes {
    svc, err := a.GetServiceByRoute(route.Tag, svcTag)
    if err != nil {
      continue
    }

    return svc, nil
  }

  return nil, errors.New("ServiceNotFoundErr")
}

func (a *Agency) GetServiceByRoute(routeTag, svcTag string) (*Service, error) {
  route, err := a.GetRoute(routeTag)
  if err != nil {
    return nil, err
  }

  svc, err := route.GetService(svcTag)
  if err != nil {
    return nil, err
  }

  return svc, nil
}

func (a *Agency) GetRoutes(opts...ApiHandlerOption) ([]*Route, error) {
  aho := DefaultApiHandlerOptions

  var useCache bool = aho.UseCache
  if a.Routes == nil {
    useCache = false
  }

  if time.Now().Sub(a.cacheAge) > (a.api.cacheMaxAge*time.Second) {
    useCache = false
  }

  if useCache {
    return a.Routes, nil
  }

  resp := a.api.Get(MethodRoutes(a.Tag))
  if resp.Error() != nil {
    return nil, resp.Error()
  }

  routes := make([]string, 0)
  routesIface := make(map[string]interface{})
  err := json.Unmarshal(resp.Data, &routesIface)
  if err != nil {
    return nil, err
  }

  // test if there is only one route 
  routeList, ok := routesIface["route"]
  if ok {
    rl, ok := routeList.([]interface{})
    if !ok {
      rl = []interface{}{routeList.(map[string]interface{})}
    }

    for _, v := range rl {
      rte, ok := v.(map[string]interface{})
      if !ok {
        continue
      }
      tag, ok := utils.IfaceToString(rte["tag"])
      if ok {
        routes = append(routes, tag)
      }
    }
  }


  rtes := make([]*Route, 0)

  for _, r := range routes {
    resp := a.api.Get(MethodRouteConfig(a.Tag, r))
    if resp.Error() != nil {
      return nil, resp.Error()
    }

    rIface := make(map[string]interface{})
    err := json.Unmarshal(resp.Data, &rIface)
    if err != nil {
      return nil, err
    }

    rcfg, err := a.unmarshalRouteConfig(rIface)
    if err != nil {
      return nil, err
    }

    rtes = append(rtes, rcfg)
  }

  a.Routes = rtes
  a.cacheAge = time.Now()
  return rtes, nil 
}

func (a *Agency) GetStop(stopId string) (*Stop, error) {
  if a.Routes == nil {
    _, err := a.GetRoutes()
    if err != nil {
      return nil, errors.New("Could not get stops from routes from api")
    }
  }

  stops := make([]*Stop, 0)
  for _, route := range a.Routes {
    stops = append(stops, route.Stops...)
  }

  for _, stop := range stops {
    if stop.StopID == stopId {
      return stop, nil
    }
  }

  return nil, errors.New("StopNotFoundErr")
}

func (a *Agency) GetStopRoutes(stopId string) ([]*Route, error) {
  if a.Routes == nil {
    _, err := a.GetRoutes()
    if err != nil {
      return nil, errors.New("Could not get stops from routes from api")
    }
  }

  routes := make([]*Route, 0)
  for _, route := range a.Routes {
    stops := make([]string, 0)
    for _, stop := range route.Stops {
      stops = append(stops, stop.StopID)
    }
    if slices.Contains(stops, stopId) {
      routes = append(routes, route)
    }
  }

  return routes, nil
}

func (a *Agency) GetStopServiceRoutes(stopId string) ([]*Service, error) {
  routes, err := a.GetStopRoutes(stopId)
  if err != nil {
    return nil, err
  }

  svcRoutes := make([]*Service, 0)
  searchSvcs := make([]*Service, 0)
  for _, route := range routes {
    searchSvcs = append(searchSvcs, route.Services...)
  }

  for _, svc := range searchSvcs {
    for _, stop := range svc.Stops {
      if stop.StopID == stopId {
        svcRoutes = append(svcRoutes, svc)
      }
    }
  }

  return svcRoutes, nil
}

func (a *Agency) GetStops(opts...ApiHandlerOption) ([]*Stop, error) {
  routes, err := a.GetRoutes(opts...)
  if err != nil {
    return nil, err
  }

  stops := make([]*Stop, 0)
  for _, route := range routes {
    stopList := route.Stops
    for _, stop := range stopList {
      if !slices.Contains(stops, stop) {
        stops = append(stops, stop)
      }
    }
  }

  return stops, nil
}
