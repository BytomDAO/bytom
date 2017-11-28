package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

//Token describe the access token.
type Token struct {
	ID      string    `json:"id,omitempty"`
	Token   string    `json:"token,omitempty"`
	Type    string    `json:"type,omitempty"`
	Secret  string    `json:"secret,omitempty"`
	Created time.Time `json:"created_at,omitempty"`
}

type resp struct {
	Status string `json:"status,omitempty"`
	Msg    string `json:"msg,omitempty"`
	Data   string `json:"data,omitempty"`
}

type respToken struct {
	Status string   `json:"status,omitempty"`
	Msg    string   `json:"msg,omitempty"`
	Data   []*Token `json:"data,omitempty"`
}

func parseresp(response interface{}, pattern interface{}) error {
	data, err := base64.StdEncoding.DecodeString(response.(string))
	if err != nil {
		jww.ERROR.Println("response format error")
		return err
	}

	if err := json.Unmarshal(data, pattern); err != nil {
		jww.ERROR.Println("result not json format", err)
		return err
	}

	return nil
}

var createAccessTokenCmd = &cobra.Command{
	Use:   "create-access-token",
	Short: "Create a access token",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("create-access-token needs 1 args")
			return
		}

		var token Token
		token.ID = args[0]

		var response interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/create-access-token", &token, &response)

		var rawresp resp
		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			jww.FEEDBACK.Printf("%v\n", rawresp.Data)
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
	},
}

var listAccessTokenCmd = &cobra.Command{
	Use:   "list-access-token",
	Short: "list access tokens",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			jww.ERROR.Println("list-access-token needs 0 args")
			return
		}

		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/list-access-token", nil, &response)

		var rawresp respToken
		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			for i, v := range rawresp.Data {
				fmt.Println(i, v.Token)
			}
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
	},
}

var deleteAccessTokenCmd = &cobra.Command{
	Use:   "delete-access-token",
	Short: "delete a access token",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("delete-access-token needs 1 args")
			return
		}

		var token Token
		token.ID = args[0]

		var response interface{}

		client := mustRPCClient()
		client.Call(context.Background(), "/delete-access-token", &token, &response)

		var rawresp resp

		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			jww.FEEDBACK.Printf("%v\n", rawresp.Data)
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
	},
}

var checkAccessTokenCmd = &cobra.Command{
	Use:   "check-access-token",
	Short: "check a access token",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			jww.ERROR.Println("check-access-token needs 1 args")
			return
		}

		var token Token
		inputs := strings.Split(args[0], ":")
		token.ID = inputs[0]
		token.Secret = inputs[1]
		var response interface{}
		client := mustRPCClient()
		client.Call(context.Background(), "/check-access-token", &token, &response)

		var rawresp resp

		if err := parseresp(response, &rawresp); err != nil {
			jww.ERROR.Println("parse response error")
			return
		}

		if rawresp.Status == "success" {
			jww.FEEDBACK.Printf("%v\n", rawresp.Data)
			return
		}

		if rawresp.Status == "error" {
			jww.ERROR.Println(rawresp.Msg)
			return
		}
	},
}
