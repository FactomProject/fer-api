
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	ed "github.com/FactomProject/ed25519"
	"github.com/FactomProject/factom"
	"github.com/FactomProject/goleveldb/leveldb/errors"
	"strconv"
)


// This is a representation of the FER data.  Basically the json of this will be the factom entry content
type FEREntry struct {
	ExpirationHeight       uint32 `json:"expiration_height"`
	TargetActivationHeight uint32 `json:"target_activation_height"`
	Priority               uint32 `json:"priority"`
	TargetPrice            uint64 `json:"target_price"`
	Version                string `json:"version"`
}


//ExpirationHeight string, ActivationHeight string, Priority string, TargetPrice string
func CreateFEREntryAndReveal(ExpirationHeight string, ActivationHeight string, Priority string, TargetPrice string) (Entry string, Reveal string, targetPriceInDollars float64, newECAddress string, err error) {

	// Read the config file
	config, err := readConfigFile("FactomFER.conf")
	if ( err != nil ) {
		errorMessage := errors.New(" Could not find config file FactomFER.conf.\n A sample config file is below, create it if you wish:\n   PaymentPrivateKey = \"0000000000000000000000000000000000000000000000000000000000000000\"\n   SigningPrivateKey = \"0000000000000000000000000000000000000000000000000000000000000000\"\n   Version = \"1.0\"")
		return "", "", 0.0, "", errorMessage
	}

	// Make an Fer Entry to send along
	theFEREntry := new(FEREntry)
	theFEREntry.Version = config.Version

	// Create and format the payment private key
	var paymentPrivateKey [32]byte
	paymentBytes, err := hex.DecodeString(config.PaymentPrivateKey)

	if (err != nil) {
		return "", "", 0.0, "", errors.New("Payment private key isn't parsable")
	}
	copy(paymentPrivateKey[:], paymentBytes)
	//paymentPublicKey := new([32]byte)
	//paymentPublicKey = ed.GetPublicKey(&paymentPrivateKey)

	// Create and format the signing private key
	var signingPrivateKey [64]byte
	signingBytes, err := hex.DecodeString(config.SigningPrivateKey)
	if (err != nil) {
		return "", "", 0.0, "",errors.New("Signing private key isn't parsable")
	}
	copy(signingPrivateKey[:], signingBytes[:])
	_ = ed.GetPublicKey(&signingPrivateKey)  // Needed to format the public half of the key set

	uExpirationHeight, err := strconv.ParseUint(ExpirationHeight, 10, 32)
	if err != nil {
		fmt.Println(err)
	}
	uintValue := uExpirationHeight
	theFEREntry.ExpirationHeight = uint32(uintValue)

	uActivationHeight, err := strconv.ParseUint(ActivationHeight, 10, 32)
	if err != nil {
		fmt.Println(err)
	}
	uintValue = uActivationHeight
	theFEREntry.TargetActivationHeight = uint32(uintValue)

	uPriority, err := strconv.ParseUint(Priority, 10, 32)
	if err != nil {
		fmt.Println(err)
	}
	uintValue = uPriority
	theFEREntry.Priority = uint32(uintValue)

	uTargetPrice, err := strconv.ParseUint(TargetPrice, 10, 64)
	if err != nil {
		fmt.Println(err)
	}
	uintValue = uTargetPrice
	theFEREntry.TargetPrice = uintValue

	entryJson, err := json.Marshal(theFEREntry)

	if err != nil {
		return "", "", 0.0, "", errors.New("Could not marshal the data into an FEREntry")
	}

	// Create the factom entry with the signing private key
	signingSignature := ed.Sign(&signingPrivateKey, entryJson)


	// Make a new factom entry and populate it
	e := new(factom.Entry)
	e.ChainID = "111111118d918a8be684e0dac725493a75862ef96d2d3f43f84b26969329bf03"
	//chain name: echo -n "This chain contains messages which coordinate the FCT to EC conversion rate amongst factomd nodes." | factom-cli addchain -e "FCT EC Conversion Rate Chain" -e "1950454129" EC2DKSYyRcNWf7RS963VFYgMExoHRYLHVeCfQ9PGPmNzwrcmgm2r
	e.ExtIDs = append(e.ExtIDs, signingSignature[:])
	e.Content = entryJson

   
	a, err := factom.MakeECAddress(paymentPrivateKey[0:32])
	if err != nil {
		fmt.Println(err)
		return
	}
	// Create the compose and the reveal
	//entryCommitJson, err := factom.ComposeEntryCommit(paymentPublicKey, signingSignature, e)
	entryCommitJson, err := factom.ComposeEntryCommit(e, a )
	if err != nil { return "", "", 0.0, "", err }
	revealJson, err := factom.ComposeEntryReveal(e)
	if err != nil { return "", "", 0.0, "", err }

	impliedFctPrice := float64(0.0)
	if (theFEREntry.TargetPrice != 0) {
		impliedFctPrice = 100000 / float64(theFEREntry.TargetPrice)
	} else {
		return "", "", 0.0, "", errors.New("Trying to set targetPrice to 0!")
	}
	commitResp, err := factom.EncodeJSONString(entryCommitJson)
	revealResp, err := factom.EncodeJSONString(revealJson)
	return commitResp, revealResp, impliedFctPrice, a.PubString(), nil
}



func GetCurlOutputForComposition(entryCommitJson string, revealJson string, targetPriceInDollars float64, ECAddress string) (output string){

	var buffer bytes.Buffer

	entry := fmt.Sprintf("    curl -i -X POST -H 'Content-Type: application/json' -d '%s' localhost:8088/v2\n", string(entryCommitJson))
	reveal := fmt.Sprintf("    curl -i -X POST -H 'Content-Type: application/json' -d '%s' localhost:8088/v2\n", string(revealJson))
	pricePerDollar := fmt.Sprintf("$%.2f", targetPriceInDollars)

	// Make the output file and print to the screen
	buffer.WriteString("***************************************************************************************************\n")
	buffer.WriteString("*\n")
	buffer.WriteString("*   WARNING:  You a making an FERChain entry with the following data:\n")
	buffer.WriteString("*\n")
	buffer.WriteString("*      ")
	buffer.WriteString(entryCommitJson)
	buffer.WriteString("\n")
	buffer.WriteString("*   Implied factoid price:\n")
	buffer.WriteString("*\n")
	buffer.WriteString("*      ")
	buffer.WriteString(pricePerDollar)
	buffer.WriteString("\n")
	buffer.WriteString("***************************************************************************************************\n")
	buffer.WriteString("\n")
	buffer.WriteString("Entry Credit Address that pays for this Entry: ")
	buffer.WriteString(ECAddress)
	buffer.WriteString("\n")
	buffer.WriteString("***************************************************************************************************\n")
	buffer.WriteString("\n")
	buffer.WriteString( entry)
	buffer.WriteString("\n")
	buffer.WriteString( reveal)
	buffer.WriteString("\n")

	return buffer.String()
}