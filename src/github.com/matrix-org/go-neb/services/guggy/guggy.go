package services

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/matrix-org/go-neb/matrix"
	"github.com/matrix-org/go-neb/plugin"
	"github.com/matrix-org/go-neb/types"
	"math"
	"net/http"
	"strings"
)

type guggyQuery struct {
	// "mp4" or "gif"
	Format string `json:"format"`
	// Query sentence
	Sentence string `json:"sentence"`
}

type guggyGifResult struct {
	ReqID  string  `json:"reqId"`
	GIF    string  `json:"gif"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type guggyService struct {
	types.DefaultService
	id            string
	serviceUserID string
	APIKey        string `json:"api_key"`
}

func (s *guggyService) ServiceUserID() string { return s.serviceUserID }
func (s *guggyService) ServiceID() string     { return s.id }
func (s *guggyService) ServiceType() string   { return "guggy" }

func (s *guggyService) Plugin(client *matrix.Client, roomID string) plugin.Plugin {
	return plugin.Plugin{
		Commands: []plugin.Command{
			plugin.Command{
				Path: []string{"guggy"},
				Command: func(roomID, userID string, args []string) (interface{}, error) {
					return s.cmdGuggy(client, roomID, userID, args)
				},
			},
		},
	}
}
func (s *guggyService) cmdGuggy(client *matrix.Client, roomID, userID string, args []string) (interface{}, error) {
	// only 1 arg which is the text to search for.
	querySentence := strings.Join(args, " ")
	gifResult, err := s.text2gifGuggy(querySentence)
	if err != nil {
		return nil, err
	}

	if gifResult.GIF == "" {
		return matrix.TextMessage{
			MsgType: "m.text.notice",
			Body:    "No GIF found!",
		}, nil
	}

	mxc, err := client.UploadLink(gifResult.GIF)
	if err != nil {
		return nil, err
	}

	return matrix.ImageMessage{
		MsgType: "m.image",
		Body:    querySentence,
		URL:     mxc,
		Info: matrix.ImageInfo{
			Height:   uint(math.Floor(gifResult.Height)),
			Width:    uint(math.Floor(gifResult.Width)),
			Mimetype: "image/gif",
		},
	}, nil
}

// text2gifGuggy returns info about a gif
func (s *guggyService) text2gifGuggy(querySentence string) (*guggyGifResult, error) {
	log.Info("Transforming to GIF query ", querySentence)

	client := &http.Client{}

	var query guggyQuery
	query.Format = "gif"
	query.Sentence = querySentence

	reqBody, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(reqBody)

	req, err := http.NewRequest("POST", "https://text2gif.guggy.com/guggify", reader)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("apiKey", s.APIKey)

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var result guggyGifResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func init() {
	types.RegisterService(func(serviceID, serviceUserID, webhookEndpointURL string) types.Service {
		return &guggyService{
			id:            serviceID,
			serviceUserID: serviceUserID,
		}
	})
}
