package api

import (
	"errors"

	"github.com/lcyvin/go-umoparse/internal/utils"
)

type Route struct {
  Title       string
  ShortTitle  string
  Tag         string
  Services    []*Service
  Stops       []*Stop
  api         *ApiHandler
  agency      *Agency
}

func (r *Route) GetService(tag string) (*Service, error) {
  if r.Services == nil {
    return nil, errors.New("NilServicesErr")
  }

  for _, v := range r.Services {
    if v.Tag == tag {
      return v, nil
    }
  }

  return nil, errors.New("ServiceNotFound")
}

func (r *Route) GetStop(stopId string) (*Stop, error) {
  if r.Stops == nil {
    return nil, errors.New("NilStopsErr")
  }

  for _, v := range r.Stops {
    if v.StopID == stopId {
      return v, nil
    }
  }

  return nil, errors.New("StopNotFound")
}

func (r *Route) GetStopByTag(stopTag string) (*Stop, error) {
  if r.Stops == nil {
    return nil, errors.New("NilStopsErr")
  }

  for _, v := range r.Stops {
    if v.Tag == stopTag {
      return v, nil
    }
  }

  return nil, errors.New("StopNotFound")
}

func (a *Agency) unmarshalRouteConfig(v interface{}) (*Route, error) {
  iface, ok := v.(map[string]interface{})
  if !ok {
    return nil, errors.New("could not cast input to map[string]interface{}")
  }

  r := &Route{
    api: a.api,
    agency: a,
  }
  svcs := make([]*Service, 0)
  stops := make([]*Stop, 0)
  // extract our stops first
  rteIface, ok := iface["route"].(map[string]interface{})
  if !ok {
    return nil, errors.New("could not get route from response")
  }
  tag, ok := utils.IfaceToString(rteIface["tag"])
  if ok {
    r.Tag = tag
  }

  title, ok := utils.IfaceToString(rteIface["title"])
  if ok {
    r.Title = title
  }

  sTitle, ok := utils.IfaceToString(rteIface["shortTitle"])
  if ok {
    r.ShortTitle = sTitle
  } else {
    r.ShortTitle = title
  }

  stopList, ok := rteIface["stop"].([]interface{})
  
  if !ok {
    return nil, errors.New("Could not get list of stops from routeConfig")
  }

  for _, protoStop := range stopList {
    s := &Stop{
      api: a.api,
      agency: a,
    }
    err := unmarshalRouteStop(s, protoStop)
    if err != nil {
      return nil, err
    }

    stops = append(stops, s)
  }
  r.Stops = stops

  var svcList []interface{}
  svcList, ok = rteIface["direction"].([]interface{})
  if !ok {
    singleSvc, ok := rteIface["direction"].(map[string]interface{})
    if !ok {
      return nil, errors.New("Could not get service routes from routeConfig")
    }
    svcList = []interface{}{singleSvc}
  }

  for _, svcProto := range svcList {
    svc := &Service{
      api: r.api,
      agency: r.agency,
      route: r,
    }
    err := unmarshalServiceRoute(svc, svcProto)
    if err != nil {
      return nil, err
    }
    svcs = append(svcs, svc)
  }

  r.Services = svcs
  return r, nil
}

func unmarshalServiceRoute(s *Service, v interface{}) error {
  svc, ok := v.(map[string]interface{})
  if !ok {
    return errors.New("Could not unmarshal service to map[string]interface{}")
  }

  tag, ok := utils.IfaceToString(svc["tag"])
  if ok {
    s.Tag = tag
  }

  name, ok := utils.IfaceToString(svc["name"])
  if ok {
    s.Name = name
  }

  title, ok := utils.IfaceToString(svc["title"])
  if ok {
    s.Title = title
  }

  ufui, ok := utils.IfaceToBool(svc["useForUI"])
  if ok {
    s.UseForUI = ufui
  }

  stops := make([]*Stop, 0)
  stopList, ok := svc["stop"].([]interface{})
  if !ok {
    return errors.New("Could not retrieve stop list from service route "+tag)
  }

  for _,stopProto := range stopList {
    stopIface, ok := stopProto.(map[string]interface{})
    if !ok {
      continue
    }

    stopTag, ok := utils.IfaceToString(stopIface["tag"])
    if ok {
      stop, err := s.route.GetStopByTag(stopTag)
      if err != nil {
        continue
      }
      stops = append(stops, stop)
    }
  }

  return nil
}

func unmarshalRouteStop(s *Stop, v interface{}) error {
  stop, ok := v.(map[string]interface{})
  if !ok {
    return errors.New("Could not marshal stop to map[string]interface{}")
  }

  tag, ok := utils.IfaceToString(stop["tag"])
  if ok {
    s.Tag = tag
  }

  id, ok := utils.IfaceToString(stop["stopId"])
  if ok {
    s.StopID = id
  }

  title, ok := utils.IfaceToString(stop["title"])
  if ok {
    s.Title = title
  }

  stitle, ok := utils.IfaceToString(stop["shortTitle"])
  if ok {
    s.ShortTitle = stitle
  }

  lon, ok := utils.IfaceToFloat(stop["lon"])
  if ok {
    s.Longitude = lon
  }

  lat, ok := utils.IfaceToFloat(stop["lat"])
  if ok {
    s.Latitude = lat
  }

  return nil
}
