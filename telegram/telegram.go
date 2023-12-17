package telegram

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
)

func SendTelegramMessage(message string) error {
	var telegramApi = "https://api.telegram.org/botXXXXXXXXXX:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/sendMessage"
	response, err := http.PostForm(telegramApi, url.Values{"chat_id": {"-873621661"}, "text": {message}, "parse_mode": {"HTML"}})
	if err != nil {
		return fmt.Errorf("failed to trigger telegram hook: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Errorf("failed to close request body: %v", err)
		}
	}(response.Body)

	bodyBytes, errRead := io.ReadAll(response.Body)
	if errRead != nil {
		return fmt.Errorf("failed to read response body: %v", errRead)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("failed to send request, status code: %d, content: %s", response.StatusCode, bodyBytes)
	}

	return nil
}
