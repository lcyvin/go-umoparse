package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lcyvin/go-umoparse/internal/utils"
)

const API_URI string = "https://retro.umoiq.com/service/publicJSONFeed"
var DefaultApiHandlerOptions *ApiHandlerOptions = &ApiHandlerOptions{
  UseCache: true,
  CacheMaxAge: 60,
}

var DefaultApiHandler *ApiHandler = NewApiHandler(&GetConfig{
  Timeout: 30,
  RetryLimit: 0,
  RetryDelay: 100,
  CustomHeaders: nil,
  Context: context.TODO(),
})

type ApiOption func(*ApiHandler)

func WithMaxCacheAge(seconds int) ApiOption {
  return func(a *ApiHandler) {
    t := time.Duration(seconds) * time.Second
    a.cacheMaxAge = t
  }
}

func WithHttpClient(c *http.Client) ApiOption {
  return func(a *ApiHandler) {
    a.c = c
  }
}

type ApiHandler struct {
  cfg         *GetConfig
  // cache to hold retrieved agencies to prevent
  // redundant requests to the API
  agencies    []*Agency
  cacheAge    time.Time
  cacheMaxAge time.Duration
  c           *http.Client
}

func (a *ApiHandler) Get(m ApiMethod) (*ApiResponse) {
  cfg := a.cfg
  if cfg.RetryDelay < 50 {
    return &ApiResponse{
      Response: nil,
      err: fmt.Errorf("Unable to use retry delay lower than 50ms"),
    }
  }

  if cfg.CustomHeaders == nil {
    cfg.CustomHeaders = make(map[string]string)
  }

  var resp *ApiResponse
  for i := 0; i <= 1+cfg.RetryLimit; i++ {
    cctx := cfg.Context
    var cancel context.CancelFunc
    if cfg.Timeout != 0 {
      cctx, cancel = context.WithTimeout(cfg.Context, time.Duration(cfg.Timeout)*time.Second)
      defer cancel()
    }

    resp = get(cctx, m, cfg.CustomHeaders, a.c)
    if resp.Error() != nil {
      time.Sleep(time.Duration(cfg.RetryDelay)*time.Millisecond)
      continue
    }
  }

  return resp
}

func (a *ApiHandler) unmarshalAgencies(v interface{}) ([]*Agency, error) {
  obj, ok := v.(map[string]interface{})
  if !ok {
    return nil, errors.New("invalid input")
  }

  agencyList, ok := obj["agency"].([]interface{})
  if !ok {
    return nil, errors.New("could not get agencies from response")
  }

  agencies := make([]*Agency, 0)

  for _, agencyProto := range agencyList {
    agency, ok := agencyProto.(map[string]interface{})
    if !ok {
      return nil, errors.New("Could not convert interface to object")
    }

    a := &Agency{
      api: a,
    }

    title, ok := utils.IfaceToString(agency["title"])
    if ok {
      a.Title = title
    } else {
      return nil, errors.New("Could not get title, this shouldn't happen")
    }

    tag, ok := utils.IfaceToString(agency["tag"])
    if ok {
      a.Tag = tag
    } else {
      return nil, errors.New("Could not get tag, this shouldn't happen")
    }

    sTitle, ok := utils.IfaceToString(agency["shortTitle"])
    if ok {
      a.ShortTitle = sTitle
    } else {
      a.ShortTitle = a.Title
    }

    rTitle, ok := utils.IfaceToString(agency["regionTitle"])
    if ok {
      a.RegionTitle = rTitle
    }

    agencies = append(agencies, a)
  }

  return agencies, nil
}

func (a *ApiHandler) GetAgencies(opts...ApiHandlerOption) ([]*Agency, error) {
  aho := DefaultApiHandlerOptions
  for _, opt := range opts {
    opt(aho)
  }

  var useCache bool

  if time.Now().Sub(a.cacheAge) > a.cacheMaxAge {
    useCache = false
  }

  if !aho.UseCache {
    useCache = false
  }

  if a.agencies != nil && useCache {
    return a.agencies, nil
  }

  resp := a.Get(MethodAgencyList())
  if resp.Error() != nil {
    return nil, resp.Error()
  }

  unmarshalIface := make(map[string]interface{})
  err := json.Unmarshal(resp.Data, &unmarshalIface)
  if err != nil {
    return nil, err
  }

  agencies, err := a.unmarshalAgencies(unmarshalIface)
  return agencies, nil
}

func (a *ApiHandler) GetAgency(agencyTag string, opts...ApiHandlerOption) (*Agency, error) {
  agencies, err := a.GetAgencies(opts...)
  if err != nil {
    return nil, err
  }

  for _, agency := range agencies {
    if agency.Tag == agencyTag {
      return agency, nil
    }
  }

  return nil, &ApiNotExistErr{msg: "Agency "+agencyTag+" not found."}
}

func NewApiHandler(cfg *GetConfig, opts... ApiOption) *ApiHandler {
  h := &ApiHandler{
    cfg: cfg,
    cacheMaxAge: time.Duration(3600),
    c: http.DefaultClient,
  }

  for _, opt := range opts {
    opt(h)
  }

  return h
}

type GetConfig struct {
  // set optional request timeout instead of system/api default
  Timeout       int
  // Limit total number of retries. If not set, no retries will be done
  RetryLimit    int
  // Set delay between retries, in milliseconds
  RetryDelay    int
  // Set custom headers for this request
  CustomHeaders map[string]string
  // use a custom context instead of the default context.Background()
  Context       context.Context
}

type GetOpt func(*GetConfig) 

func WithTimeout(milli int) GetOpt {
  return func(c *GetConfig) {
    c.Timeout = milli
  }
}

func WithRetryLimit(limit int) GetOpt {
  return func(c *GetConfig) {
    c.RetryLimit = limit
  }
}

func WithRetryDelay(milli int) GetOpt {
  return func(c *GetConfig) {
    c.RetryDelay = milli
  }
}

func WithContext(ctx context.Context) GetOpt {
  return func(c *GetConfig) {
    c.Context = ctx
  }
}

func WithHeaders(headers map[string]string) GetOpt {
  return func(c *GetConfig) {
    c.CustomHeaders = headers
  }
}

type ApiMethod func() (string)

func MethodRoutes(agency string) ApiMethod {
  return func() (string) {
    return fmt.Sprintf("?command=routeList&a=%s", agency)
  }
}

func MethodRouteConfig(agency, route string) ApiMethod {
  return func() string {
    return fmt.Sprintf("?command=routeConfig&a=%s&r=%s&verbose", agency, route)
  }
}

func MethodAgencyList() ApiMethod {
  return func() string {
    return "?command=agencyList"
  }
}

func MethodSchedule(agency, route string) ApiMethod {
  return func() string {
    return fmt.Sprintf("?command=schedule&a=%s&r=%s", agency, route)
  }
}

func MethodVehicleLocations(agency, route, time string) ApiMethod {
  return func() string {
    return fmt.Sprintf("?command=vehicleLocations&a=%s&r=%s&t=%s", agency, route, time)
  }
}

func MethodVehicleLocation(agency, vehicleId string) ApiMethod {
  return func() string {
    return fmt.Sprintf("?command=vehicleLocation&a=%s&v=%s", agency, vehicleId)
  }
}

func MethodPredictions(agency, stopId, route string) ApiMethod {
  return func() string {
    apiCmd := fmt.Sprintf("?command=predictions&a=%s&stopId=%s", agency, stopId)
    if route != "" {
      apiCmd = apiCmd + "&routeTag=" + route
    }

    return apiCmd
  }
}

type NoResponseError struct {
  msg string
}

func (e *NoResponseError) Error() string {
  return e.msg
}

type ApiMethodResponse interface {
  Reader() (io.Reader, error)
  Error()  (error)
}

type ApiResponse struct {
  Response *http.Response
  err      error
  Data     []byte
}

func (ar *ApiResponse) Reader() (io.Reader) {
  return bytes.NewReader(ar.Data)
}

func (ar *ApiResponse) Error() error {
  return ar.err
}

func get(ctx context.Context, m ApiMethod, headers map[string]string, c *http.Client) (*ApiResponse) {
  apiCmd := m()
  apiResp := &ApiResponse{}
  
  request, err := http.NewRequestWithContext(ctx, http.MethodGet, API_URI+apiCmd, nil)
  if err != nil {
    return &ApiResponse{err: err}
  }

  if headers != nil || len(headers) > 0 {
    for k,v := range headers {
      request.Header.Set(k, v)
    }
  }

  resp, err := c.Do(request)
  apiResp.Response = resp
  apiResp.err = err
  if err != nil {
    return apiResp
  }

  data, err := io.ReadAll(resp.Body)
  if err != nil {
    apiResp.err = err
  }

  apiResp.Data = data

  return apiResp
}

// Default client Get, to use custom request configuration create a new
// api handler and call Get from that.
func Get(m ApiMethod) *ApiResponse {
  return DefaultApiHandler.Get(m)
}
