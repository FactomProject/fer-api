// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/FactomProject/factom"
	"github.com/FactomProject/web"
	"io/ioutil"
)

var (
	webServer *web.Server
	pflag = flag.Int("p", 9999, "set the port to host the wsapi")
)

const httpBad = 400

func handleV2Error(ctx *web.Context, j *factom.JSON2Request, err *factom.JSONError) {
	resp := factom.NewJSON2Response()
	if j != nil {
		resp.ID = j.ID
	} else {
		resp.ID = nil
	}
	resp.Error = err

	ctx.WriteHeader(httpBad)
	ctx.Write([]byte(resp.String()))
}

func newInvalidRequestError() *factom.JSONError {
	return factom.NewJSONError(-32600, "Invalid Request", nil)
}
func newCustomInternalError(data interface{}) *factom.JSONError {
	return factom.NewJSONError(-32603, "Internal error", data)
}

func handleV2(ctx *web.Context) {
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		handleV2Error(ctx, nil, newInvalidRequestError())
		return
	}

	j, err := factom.ParseJSON2Request(string(body))
	if err != nil {
		handleV2Error(ctx, nil, newInvalidRequestError())
		return
	}

	jsonResp, jsonError := handleV2Request(j)

	if jsonError != nil {
		handleV2Error(ctx, j, jsonError)
		return
	}

	ctx.Write([]byte(jsonResp.String()))
}

func handleV2Request(j *factom.JSON2Request) (*factom.JSON2Response, *factom.JSONError) {
	var resp interface{}
	var jsonError *factom.JSONError
	params := []byte(j.Params)

	resp, jsonError = handleGenerateECAddress(params)

	if jsonError != nil {
		return nil, jsonError
	}

	fmt.Printf("API V2 method: <%v>  parameters: %s\n", j.Method, params)

	jsonResp := factom.NewJSON2Response()
	jsonResp.ID = j.ID
	if b, err := json.Marshal(resp); err != nil {
		return nil, newCustomInternalError(err.Error())
	} else {
		jsonResp.Result = b
	}

	return jsonResp, nil
}

func handleGenerateECAddress(params []byte) (interface{}, *factom.JSONError) {
	entry, reveal, targetPriceInDollars, ecAddress, err := CreateFEREntryAndReveal()
	if (err != nil) {
		fmt.Println("Error: ", err)
		return nil, newCustomInternalError(err.Error())
	}

	compositionString := GetCurlOutputForComposition(entry, reveal, targetPriceInDollars, ecAddress)

	fmt.Println(compositionString)

	_, err = WriteToFile("FERComposeCurls.dat", compositionString)
	if (err != nil) {
		fmt.Println("Error: ", err)
		return nil, newCustomInternalError(err.Error())
	}

	return

	r := new(addressResponse)
	r.Public = "Good stuff boi"
	resp := r

	return resp, nil
}

type addressResponse struct {
	Public string `json:"10/10 "`
}

// The main reads the config file, gets values from the command line for the FEREntry,
// and then makes a curl commit and reveal string which it sends to a file.
func main() {
	port := *pflag

	webServer = web.NewServer()
	webServer.Post("/change-price", handleV2)
	webServer.Run(fmt.Sprintf(":%d", port))

	//entry, reveal, targetPriceInDollars, ecAddress, err := CreateFEREntryAndReveal()
	//if (err != nil) {
	//	fmt.Println("Error: ", err)
	//	return
	//}
	//
	//compositionString := GetCurlOutputForComposition(entry, reveal, targetPriceInDollars, ecAddress)
	//
	//fmt.Println(compositionString)
	//
	//_, err = WriteToFile("FERComposeCurls.dat", compositionString)
	//if (err != nil) {
	//	fmt.Println("Error: ", err)
	//	return
	//}
	//
	//return
}
