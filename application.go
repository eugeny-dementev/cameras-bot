package main

type Application struct {
  config Config
}

func (a *Application) Init() error {
  a.config = Config{}
  err := a.config.Setup()
  if err != nil {
    return err
  }

  return nil
}
