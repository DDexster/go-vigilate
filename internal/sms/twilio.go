package sms

import (
	"encoding/json"
	"fmt"
	"github.com/DDexster/go-vigilate/internal/config"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"log"
)

func SendTextTwilio(to, message string, app *config.AppConfig) error {
	token := app.PreferenceMap["twilio_auth_token"]
	sid := app.PreferenceMap["twilio_sid"]

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: sid,
		Password: token,
	})

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(app.PreferenceMap["twilio_phone_number"])
	params.SetBody(message)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		log.Println(err)
		return err
	}

	response, _ := json.Marshal(*resp)
	log.Println(fmt.Sprintf("Twilio Response: %s", string(response)))
	return nil
}
