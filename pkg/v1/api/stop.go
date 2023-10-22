package api

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/lcyvin/go-umoparse/internal/utils"
)

type Stop struct {
  StopID        string
  Tag           string
  Title         string
  ShortTitle    string
  Longitude     float64
  Latitude      float64
  Predictions   []*Prediction
  predictionMap map[string][]*Prediction
  cacheAge      time.Time
  api           *ApiHandler
  agency        *Agency
}

func (s *Stop) GetPredictions(opts...ApiHandlerOption) ([]*Prediction, error) {
  aho := &ApiHandlerOptions{
    UseCache: true,
  }

  for _, opt := range opts {
    opt(aho)
  }

  var cacheMaxAge int = 0
  now := time.Now()
  if aho.CacheMaxAge != 0 {
    cacheMaxAge = aho.CacheMaxAge
  }

  var useCache bool = true

  if s.Predictions == nil {
    useCache = false
  }

  if now.Sub(s.cacheAge) > (time.Duration(cacheMaxAge)*time.Second) {
    useCache = false
  }

  if aho.UseCache == false {
    useCache = false
  }

  if useCache {
    return s.Predictions, nil
  }

  data, err := s.predictionRequest("")
  predictions := make([]*Prediction, 0)

  pIface := make(map[string]interface{})
  err = json.Unmarshal(data, &pIface)
  if err != nil {
    return nil, err
  }

  // we need to test if there's only one object or an array returned,
  // because UmoIQ doesn't maintain schema when there's only one
  // item in a response
  preds, ok := pIface["predictions"]
  if !ok {
    return nil, errors.New("PredictionUnmarshalErr")
  }

  predArray, ok := preds.([]interface{})
  if !ok {
    predSingle, ok := preds.(map[string]interface{})
    if !ok {
      return nil, errors.New("PredictionUnmarshalErr")
    }

    predArray = []interface{}{predSingle}
  }
  for _, routePreds := range predArray {
    rp, ok := routePreds.(map[string]interface{})
    if !ok {
      return nil, errors.New("RoutePredictionUnmarshalErr")
    }

    _, emptyTest := utils.IfaceToString(rp["dirTitleBecauseNoPredictions"])
    if emptyTest {
      continue
    }


    svcPreds, ok := rp["direction"]
    if !ok {
      continue
    }
    
    pset, err := unmarshalPredictionServiceRoutes(svcPreds, s)
    if err != nil {
      return nil, err
    }

    predictions = append(predictions, pset...)
  }

  s.Predictions = predictions
  return predictions, nil
}

func (s *Stop) predictionRequest(routeTag string) ([]byte, error) {
  resp := s.api.Get(MethodPredictions(s.agency.Tag, s.StopID, routeTag))
  if resp.Error() != nil {
    return nil, resp.Error()
  }

  return resp.Data, nil
}
