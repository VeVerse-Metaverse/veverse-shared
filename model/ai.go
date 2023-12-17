package model

import (
	"bytes"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"context"
	glContext "dev.hackerman.me/artheon/veverse-shared/context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/polly"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
)

type AiSimpleFsmStatesRequest struct {
	Actions     []string `json:"actions"`
	Lines       int      `json:"lines"`
	Context     string   `json:"context"`
	Subjects    []string `json:"subjects"`
	Objects     []string `json:"objects"`
	Environment string   `json:"environment"`
}

type AiSimpleFsmStatesRequestV2 struct {
	States    int    `json:"n"` // States number
	Context   string `json:"c"` // Context
	Locations []struct {
		Name        string `json:"n"` // Name
		Description string `json:"d"` // Description
		Links       []struct {
			Name        string `json:"n"` // Name
			Description string `json:"d"` // Description
			Target      string `json:"t"` // Target location
			Object      string `json:"o"` // Object to use to get to target location, if any
		} `json:"l"` // Links to other locations
		Entities struct {
			NPCs []struct {
				Name        string `json:"n"` // Name
				Description string `json:"d"` // Description
			} `json:"n"` // NPCs in this location
			Players []struct {
				Name        string `json:"n"` // Name
				Description string `json:"d"` // Description
			} `json:"p"` // Players in this location
			Objects []struct {
				Name        string `json:"n"` // Name
				Description string `json:"d"` // Description
			} `json:"o"` // Objects in this location
		} `json:"e"` // Entities in this location
	} `json:"l"` // Locations
}

type AiTextToSpeechRequest struct {
	Engine          string   `json:"engine"`
	LanguageCode    string   `json:"languageCode"`
	LexiconNames    []string `json:"lexiconNames"`
	OutputFormat    string   `json:"outputFormat"`
	SampleRate      string   `json:"sampleRate"`
	SpeechMarkTypes []string `json:"speechMarkTypes"`
	Text            string   `json:"text"`
	TextType        string   `json:"textType"`
	VoiceId         string   `json:"voiceId"`
}

var (
	SupportedAiTextToSpeechEngines = map[string]bool{
		"neural":   true,
		"standard": true,
	}

	SupportedAiTextToSpeechLanguageCodes = map[string]bool{
		"en-US": true,
	}

	SupportedAiTextToSpeechOutputFormats = map[string]bool{
		"json":       true,
		"mp3":        true,
		"ogg_vorbis": true,
		"pcm":        true,
	}

	SupportedAiTextToSpeechSampleRates = map[string]bool{
		"8000":  true,
		"16000": true,
		"22050": true,
		"24000": true,
	}

	SupportedAiTextToSpeechSpeechMarkTypes = map[string]bool{
		"sentence": true,
		"ssml":     true,
		"viseme":   true,
		"word":     true,
	}

	SupportedAiTextToSpeechTextTypes = map[string]bool{
		"ssml": true,
		"text": true,
	}

	SupportedAiTextToSpeechVoiceIds = map[string]bool{
		"Salli":    true,
		"Kimberly": true,
		"Kendra":   true,
		"Joanna":   true,
		"Ivy":      true,
		"Ruth":     true,
		"Kevin":    true,
		"Matthew":  true,
		"Justin":   true,
		"Joey":     true,
		"Stephen":  true,
	}
)

const (
	AiTextToSpeechRequestDefaultEngine       = "neural"
	AiTextToSpeechRequestDefaultLanguageCode = "en-US"
	AiTextToSpeechRequestDefaultOutputFormat = "mp3"
	AiTextToSpeechRequestDefaultSampleRate   = "24000"
	AiTextToSpeechRequestDefaultVoiceId      = "Joanna"
	AiTextToSpeechRequestDefaultTextType     = "text"
)

func RequestPollyAudio(tts *polly.Polly, r AiTextToSpeechRequest) (audio io.ReadCloser, err error) {
	input := &polly.SynthesizeSpeechInput{
		Engine:          aws.String(r.Engine),
		LanguageCode:    aws.String(r.LanguageCode),
		OutputFormat:    aws.String(r.OutputFormat),
		SampleRate:      aws.String(r.SampleRate),
		SpeechMarkTypes: aws.StringSlice(r.SpeechMarkTypes),
		Text:            aws.String(r.Text),
		TextType:        aws.String(r.TextType),
		VoiceId:         aws.String(r.VoiceId),
	}

	result, err := tts.SynthesizeSpeech(input)
	if err != nil {
		return nil, err
	}

	return result.AudioStream, nil
}

func RequestGoogleCloudTtsAudio(client *texttospeech.Client, r AiTextToSpeechRequest) (audio io.ReadCloser, err error) {
	input := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: r.Text,
			},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: r.LanguageCode,
			Name:         r.VoiceId,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}

	resp, err := client.SynthesizeSpeech(context.Background(), input)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(resp.AudioContent)), nil
}

type CreateAiSimpleFsmRequest struct {
	Text string `json:"text"`
}

func CreateAiSimpleFsmScript(ctx context.Context, requester *User, request CreateAiSimpleFsmRequest) (err error) {
	if requester == nil {
		return ErrNoRequester
	}

	if !requester.IsAdmin {
		return ErrNoPermission
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return ErrNoDatabase
	}

	q := `insert into ai_simple_fsm_script (id, text) values (gen_random_uuid(), $1);`

	_, err = db.Exec(ctx, q, request.Text)
	if err != nil {
		return err
	}

	return nil
}
