package amqp

import "time"

// @title System controller
// @version Mark 1
// @description System management service

type EmergencyCommand struct {
	// Confirmation code, written on a piece of paper under the main boss's keyboard
	ConfirmationCode string `json:"confirmation_code" example:"endgame" validate:"required"`

	Timeout int `json:"timeout" description:"Time in seconds until execution" example:"3" validate:"min=3,required"`
}

// EmergencyButton asyncApi
// @summary complete data destruction
// @description Deleting all existing data and resetting the system
// @payload amqp.EmergencyCommand
// @queue emergency
// @tags emergency
// @contentType application/json
func EmergencyButton(payload EmergencyCommand) {
	if payload.ConfirmationCode != "1234" {
		panic("incorrect confirmation code")
		return
	}
	time.Sleep(time.Duration(payload.Timeout) * time.Second)

	DeleteAllData()
	// todo `sudo rm -rf /`
}

// CancelButton asyncApi
// @summary cancel data destruction
// @description Cancelling the deletion of the system, can only be invoked until the deletion call timeout has expired
// @queue emergency
// @tags emergency
func CancelButton() {
	// todo: add cancel data deletion
}

// DeleteAllData asyncApi
// @summary Command to queue subscribers to delete all data
// @queue subsystems.commands
// @header overwrite_count: description='how many times the data needs to be overwritten';type=int;example=5;
// @tags commands
// @operation subscribe
func DeleteAllData() {
	// todo: add publish data
}
