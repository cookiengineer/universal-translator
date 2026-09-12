// Package translator implements the background translation service that owns
// the native Bergamot engine and serialises access to it.
package translator

import (
	"errors"

	"universal-translator/adapters/bergamot"
)

type commandKind int

const (
	commandLoad commandKind = iota
	commandTranslate
	commandCancel
)

type command struct {
	kind             commandKind
	firstConfigPath  string
	secondConfigPath string
	sourceText       string
}

type translationOutcome struct {
	text string
	err  error
}

// Service serialises access to the native engine through a single worker
// goroutine. Results and errors are reported through the callbacks supplied at
// construction; the callbacks may be invoked from the worker goroutine.
type Service struct {
	engine *bergamot.Engine

	commandChannel chan command
	closeChannel   chan struct{}

	onResult func(string)
	onError  func(error)
	onBusy   func(bool)
	onLoaded func()
}

// NewService creates a translation service and starts its worker goroutine.
// The callbacks may be invoked from the worker goroutine. onLoaded is called
// after each model load attempt completes, before any load error is reported.
func NewService(numWorkers int, cacheSize int, onResult func(string), onError func(error), onBusy func(bool), onLoaded func()) (*Service, error) {
	engine, err := bergamot.NewEngine(numWorkers, cacheSize)
	if err != nil {
		return nil, err
	}

	service := &Service{
		engine:         engine,
		commandChannel: make(chan command),
		closeChannel:   make(chan struct{}),
		onResult:       onResult,
		onError:        onError,
		onBusy:         onBusy,
		onLoaded:       onLoaded,
	}

	go service.run()
	return service, nil
}

// Load requests the engine to load the model(s) for a language direction.
func (service *Service) Load(firstConfigPath string, secondConfigPath string) {
	service.commandChannel <- command{
		kind:             commandLoad,
		firstConfigPath:  firstConfigPath,
		secondConfigPath: secondConfigPath,
	}
}

// Translate requests a translation of the given source text.
func (service *Service) Translate(sourceText string) {
	service.commandChannel <- command{kind: commandTranslate, sourceText: sourceText}
}

// Cancel interrupts the in-progress translation, if any.
func (service *Service) Cancel() {
	service.commandChannel <- command{kind: commandCancel}
}

// Close shuts down the worker and releases the native engine.
func (service *Service) Close() {
	close(service.closeChannel)
}

func (service *Service) run() {
	defer service.engine.Close()

	var firstModel *bergamot.Model
	var secondModel *bergamot.Model
	var translationDone chan translationOutcome

	for {
		select {
		case next := <-service.commandChannel:
			switch next.kind {
			case commandLoad:
				service.engine.Cancel()
				translationDone = nil
				service.unloadModels(firstModel, secondModel)
				firstModel, secondModel = nil, nil

				service.onBusy(true)
				loadedFirst, loadedSecond, err := service.loadModels(next)
				service.onBusy(false)
				if service.onLoaded != nil {
					service.onLoaded()
				}

				if err != nil {
					service.onError(err)
					continue
				}
				firstModel, secondModel = loadedFirst, loadedSecond

			case commandTranslate:
				if firstModel == nil {
					service.onError(errors.New("no translation model is loaded"))
					continue
				}
				service.onBusy(true)
				translationDone = make(chan translationOutcome, 1)
				go service.runTranslation(firstModel, secondModel, next.sourceText, translationDone)

			case commandCancel:
				service.engine.Cancel()
				translationDone = nil
				service.onBusy(false)
			}

		case outcome := <-translationDone:
			translationDone = nil
			service.onBusy(false)
			if outcome.err != nil {
				service.onError(outcome.err)
			} else {
				service.onResult(outcome.text)
			}

		case <-service.closeChannel:
			service.engine.Cancel()
			service.unloadModels(firstModel, secondModel)
			return
		}
	}
}

func (service *Service) loadModels(command command) (*bergamot.Model, *bergamot.Model, error) {
	first, err := service.engine.LoadModel(command.firstConfigPath)
	if err != nil {
		return nil, nil, err
	}

	var second *bergamot.Model
	if command.secondConfigPath != "" {
		second, err = service.engine.LoadModel(command.secondConfigPath)
		if err != nil {
			first.Close()
			return nil, nil, err
		}
	}

	return first, second, nil
}

func (service *Service) runTranslation(first *bergamot.Model, second *bergamot.Model, sourceText string, done chan<- translationOutcome) {
	var text string
	var err error

	if second != nil {
		text, err = service.engine.Pivot(first, second, sourceText)
	} else {
		text, err = service.engine.Translate(first, sourceText)
	}

	done <- translationOutcome{text: text, err: err}
}

func (service *Service) unloadModels(first *bergamot.Model, second *bergamot.Model) {
	if first != nil {
		first.Close()
	}
	if second != nil {
		second.Close()
	}
}
