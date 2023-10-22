package api

import (
	"errors"
	"time"

	"github.com/lcyvin/go-umoparse/internal/utils"
)

type Prediction struct {
  // AKA "Direction", the particular route 
  // service for this predicted arrival
  Service           *Service
  // The particular route for this predicted
  // arrival
  Route             *Route
  // the ETA for this predicted arrival. This may 
  // be from live data, or from a pre-set schedule.
  // check "ScheduleBased" to determine what the source
  // of the data is.
  Eta               time.Time
  // Minutes until the estimated arrival. UmoIQ recommends
  // using this value for user-facing data, rather than Seconds
  Minutes           int64
  // Seconds until the estimated arrival. UmoIQ recommends
  // using minutes instead of this value, and instead using
  // this value to determine when to update a given prediction
  Seconds           int64
  // Per UmoIQ: only provided for predictions on the toronto
  // TTC agency. 
  Branch            string
  // Signifies that an ETA is less accurate due to a layover 
  // within the route (eg, an operator change) that will cause
  // the vehicle to dwell at a stop for several minutes. 
  AffectedByLayover bool
  // If a route begins its trip at a given stop, it may 
  // set this value in order to signify that the vehicle 
  // will leave the stop at the estimated time, rather than
  // arriving at that time.
  IsDeparture       bool
  // Only present if the tripID is set by the agency, refers to
  // a specific vehicle's designated trip along a route/serviceRoute
  TripTag           string
  // Signals if the prediction is solely based on pre-set scheduling,
  // or using live positioning data.
  ScheduleBased     bool
  // Signals that the vehicle has been travelling slower than expected
  // for the past several minutes, indicating possible traffic
  // or other delays.
  Delayed           bool
  // When this prediction was made, can be used for determining when to
  // refresh a prediction or predictions
  PredictionTime    time.Time
  stop              *Stop
  agency            *Agency
}

func (p *Prediction) unmarshalPrediction(v interface{}) (error) {
  pIface, ok := v.(map[string]interface{})
  if !ok {
    return errors.New("Could not unmarshal prediction to map[string]interface{}")
  }

  svcTag, ok := utils.IfaceToString(pIface["dirTag"])
  if !ok {
    return errors.New("Could not get service from prediction")
  }
  svc, err := p.agency.GetService(svcTag)
  if err != nil {
    return err
  }

  p.Service = svc
  p.Route = svc.route

  etaVal, ok := utils.IfaceToInt(pIface["epochTime"])
  if !ok {
    return errors.New("Could not get prediction time from response")
  }
  p.Eta = time.UnixMilli(int64(etaVal))

  min, ok := utils.IfaceToInt(pIface["minutes"])
  if ok {
    p.Minutes = int64(min)
  }

  sec, ok := utils.IfaceToInt(pIface["seconds"])
  if ok {
    p.Seconds = int64(sec)
  }

  branch, ok := utils.IfaceToString(pIface["branch"])
  if ok {
    p.Branch = branch
  }

  layover, _ := utils.IfaceToBool(pIface["affectedByLayover"])
  p.AffectedByLayover = layover

  depart, _ := utils.IfaceToBool(pIface["isDeparture"])
  p.IsDeparture = depart

  ttag, ok := utils.IfaceToString(pIface["tripTag"])
  if ok {
    p.TripTag = ttag
  }

  sched, _ := utils.IfaceToBool(pIface["isScheduleBased"])
  p.ScheduleBased = sched

  delayed, _ := utils.IfaceToBool(pIface["delayed"])
  p.Delayed = delayed

  if p.PredictionTime.IsZero() {
    p.PredictionTime = time.Now()
  }

  return nil
}

func unmarshalPredictionServiceRoutes(v interface{}, stop *Stop) ([]*Prediction, error) {
  preds := make([]*Prediction, 0)
  svcPreds, ok := v.([]interface{})
  if !ok {
    svcPredSingle, ok := v.(map[string]interface{})
    if !ok {
      return nil, errors.New("ServicePredictionsUnmarshalErr")
    }
    svcPreds = []interface{}{svcPredSingle}
  }

  for _, svc := range svcPreds {
    sp, ok := svc.(map[string]interface{})
    if !ok {
      continue
    }

    spPreds, ok := sp["prediction"].([]interface{})
    if !ok {
      spPredSingle, ok := sp["prediction"].(map[string]interface{})
      if !ok {
        continue
      }

      spPreds = []interface{}{spPredSingle}
    }

    now := time.Now()
    for _, pred := range spPreds {
      p := &Prediction{
        agency: stop.agency,
        stop: stop,
        PredictionTime: now,
      }

      err := p.unmarshalPrediction(pred)
      if err != nil {
        return nil, err
      }

      preds = append(preds, p)
    }
  }

  return preds, nil
} 
