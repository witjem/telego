package telego

import (
	"fmt"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/mymmrac/telego/api"
)

const (
	updateChanBuffer = 100

	defaultUpdateInterval = time.Second / 2 // 0.5s
	retryTimeout          = time.Second * 3 // 3s
)

const listeningForWebhookErrMsg = "Listening for webhook: %v"

// SetUpdateInterval sets interval of calling GetUpdates in GetUpdatesViaLongPulling method. Ensures that between two
// calls of GetUpdates will be at least specified time, but it could be longer.
func (b *Bot) SetUpdateInterval(interval time.Duration) {
	b.updateInterval = interval
}

// GetUpdatesViaLongPulling receive updates in chan using GetUpdates method
func (b *Bot) GetUpdatesViaLongPulling(params *GetUpdatesParams) (chan Update, error) {
	b.stopChannel = make(chan struct{})
	updatesChan := make(chan Update, updateChanBuffer)

	if params == nil {
		params = &GetUpdatesParams{}
	}

	go func() {
		for {
			select {
			case <-b.stopChannel:
				close(updatesChan)
				return
			default:
				// Continue getting updates
			}

			updates, err := b.GetUpdates(params)
			if err != nil {
				b.log.Errorf("Getting updates: %v", err)
				b.log.Errorf("Retrying to get updates in %s", retryTimeout.String())

				time.Sleep(retryTimeout)
				continue
			}

			for _, update := range updates {
				if update.UpdateID >= params.Offset {
					params.Offset = update.UpdateID + 1
					updatesChan <- update
				}
			}

			time.Sleep(b.updateInterval)
		}
	}()

	return updatesChan, nil
}

// StopGettingUpdates stop reviving updates from GetUpdatesViaLongPulling method
func (b *Bot) StopGettingUpdates() {
	close(b.stopChannel)
}

// StartListeningForWebhookTLS start server with TLS for listening for webhook
func (b *Bot) StartListeningForWebhookTLS(address, certificateFile, keyFile string) {
	go func() {
		err := b.server.ListenAndServeTLS(address, certificateFile, keyFile)
		if err != nil {
			b.log.Errorf(listeningForWebhookErrMsg, err)
		}
	}()
}

// StartListeningForWebhookTLSEmbed start server with TLS (embed) for listening for webhook
func (b *Bot) StartListeningForWebhookTLSEmbed(address string, certificateData []byte, keyData []byte) {
	go func() {
		err := b.server.ListenAndServeTLSEmbed(address, certificateData, keyData)
		if err != nil {
			b.log.Errorf(listeningForWebhookErrMsg, err)
		}
	}()
}

// StartListeningForWebhook start server for listening for webhook
func (b *Bot) StartListeningForWebhook(address string) {
	go func() {
		err := b.server.ListenAndServe(address)
		if err != nil {
			b.log.Errorf(listeningForWebhookErrMsg, err)
		}
	}()
}

// StopListeningForWebhook shutdown webhook server used in GetUpdatesViaWebhook method.
// Note: should be called only after both GetUpdatesViaWebhook and StartListeningForWebhook, etc.
func (b *Bot) StopListeningForWebhook() error {
	close(b.stopChannel)
	return b.server.Shutdown()
}

// GetUpdatesViaWebhook receive updates in chan from webhook
func (b *Bot) GetUpdatesViaWebhook(path string) (chan Update, error) {
	updatesChan := make(chan Update, updateChanBuffer)
	b.stopChannel = make(chan struct{})

	b.server.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) != path {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			b.log.Errorf("Unknown path was used in webhook: %q", ctx.Path())
			return
		}

		if method := string(ctx.Method()); method != fasthttp.MethodPost {
			err := fmt.Errorf("used invalid HTTP method: %q, required method: %q", method, fasthttp.MethodPost)
			b.respondWithError(ctx, err)

			b.log.Errorf("Webhook invalid HTTP method: %q", method)
			return
		}

		var update Update
		err := json.Unmarshal(ctx.PostBody(), &update)
		if err != nil {
			b.respondWithError(ctx, fmt.Errorf("decoding update: %w", err))

			b.log.Errorf("Webhook decoding error: %v", err)
			return
		}

		updatesChan <- update

		ctx.SetStatusCode(fasthttp.StatusOK)
	}

	go func() {
		<-b.stopChannel
		close(updatesChan)
	}()

	return updatesChan, nil
}

func (b *Bot) respondWithError(ctx *fasthttp.RequestCtx, err error) {
	errMsg, _ := json.Marshal(map[string]string{"error": err.Error()})

	ctx.SetStatusCode(fasthttp.StatusBadRequest)
	ctx.SetContentType(api.ContentTypeJSON)

	_, err = ctx.Write(errMsg)
	if err != nil {
		b.log.Error("Writing HTTP:", err)
	}
}
