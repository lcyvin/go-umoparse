package api

type ApiNotExistErr struct {
  msg string
}

func (a *ApiNotExistErr) Error() string {
  return a.msg
}

func GetAgencies() ([]*Agency, error) {
  return DefaultApiHandler.GetAgencies()
}

type ApiHandlerOptions struct {
  UseCache    bool
  CacheMaxAge int
}

type ApiHandlerOption func(*ApiHandlerOptions)

func WithoutCache() ApiHandlerOption {
  return func(a *ApiHandlerOptions) {
    a.UseCache = false
  }
}

func WithCacheMaxAge(seconds int) ApiHandlerOption {
  return func(a *ApiHandlerOptions) {
    a.CacheMaxAge = seconds
  }
}
