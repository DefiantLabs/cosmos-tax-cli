package calcfi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	parsingTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules"
	txTypes "github.com/DefiantLabs/cosmos-tax-cli/cosmos/modules/tx"
	"github.com/DefiantLabs/cosmos-tax-cli/cosmwasm/modules/wasm"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ wasm.ContractExecutionMessageHandler = &DCA575ExecutionHandler{}
	_ txTypes.CosmosMessage                = &DCA575ParserHandler{}
)

type DCA575ExecutionType string

const (
	DCA575ExecutionTypeExecuteTrigger DCA575ExecutionType = "execute_trigger"
)

const (
	// Provided as const for code clarity
	MainnetCalcfiDCACodeID575             = 575
	MainnetCalcfiDCACodeID575FriendlyName = "Calcfi DCA - 575"
)

type DCA575ExecutionHandler struct{}

func (c *DCA575ExecutionHandler) ContractFriendlyName() string {
	return MainnetCalcfiDCACodeID575FriendlyName
}

func (c *DCA575ExecutionHandler) TopLevelFieldIdentifiers() []string {
	identifierType := DCA575Type{}

	v := reflect.TypeOf(identifierType)

	reflectTags := []string{}

	for i := 0; i < v.NumField(); i++ {
		jsonTag := v.Field(i).Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		reflectTags = append(reflectTags, jsonTag)
	}

	return reflectTags
}

// Pulled from the execute schema here: https://github.com/calculated-finance/calc/blob/a526f4c6fe73cf84f391124a2ff70367ec3244d4/contracts/dca/schema/raw/execute.json
type DCA575Type struct {
	CreateVault          json.RawMessage `json:"create_vault"`
	Deposit              json.RawMessage `json:"deposit"`
	UpdateVault          json.RawMessage `json:"update_vault"`
	CancelVault          json.RawMessage `json:"cancel_vault"`
	ExecuteTrigger       *ExecuteTrigger `json:"execute_trigger"`
	UpdateConfig         json.RawMessage `json:"update_config"`
	UpdateSwapAdjustment json.RawMessage `json:"update_swap_adjustment"`
	DisburseEscrow       json.RawMessage `json:"disburse_escrow"`
	ZDelegate            json.RawMessage `json:"z_delegate"`
	Receive              json.RawMessage `json:"receive"`
}

type ExecuteTrigger struct {
	TriggerID string `json:"trigger_id"`
	Route     string `json:"route"`
}

func (c *DCA575ExecutionHandler) TopLevelIdentifierType() any {
	return any(DCA575Type{})
}

func (c *DCA575ExecutionHandler) HandlerFuncs() []func() wasm.ContractExectionParserHandler {
	return []func() wasm.ContractExectionParserHandler{
		func() wasm.ContractExectionParserHandler {
			return &DCA575ParserHandler{}
		},
	}
}

func (c *DCA575ExecutionHandler) CodeID() uint64 {
	return MainnetCalcfiDCACodeID575
}

type DCA575ParserHandler struct {
	CosmosMsgExecuteContract *wasmTypes.MsgExecuteContract
	ExecutionType            DCA575ExecutionType
}

func (c *DCA575ParserHandler) String() string {
	execTypeString := ""
	if c.ExecutionType == "" {
		execTypeString = "with unknown execution type"
	} else {
		execTypeString = fmt.Sprintf("with execution parser %s", c.ExecutionType)
	}

	return fmt.Sprintf("MsgExecuteContract: Contract %s execution parsed as %s %s", c.CosmosMsgExecuteContract.Contract, MainnetCalcfiDCACodeID575FriendlyName, execTypeString)
}

func (c *DCA575ParserHandler) HandleMsg(typeURL string, msg sdk.Msg, log *txTypes.LogMessage) error {
	executionType := DCA575Type{}

	err := json.Unmarshal(c.CosmosMsgExecuteContract.Msg, &executionType)
	if err != nil {
		return err
	}

	// Ensure that we have a non-empty execution type so we can determine the type of message and fail on unknown types that will need research/implementation
	emptyUnmarshal, err := IsEmpty(executionType)
	if err != nil {
		return errors.New("error checking for empty execution type")
	} else if emptyUnmarshal {
		errorParse := make(map[string]interface{})

		err := json.Unmarshal(c.CosmosMsgExecuteContract.Msg, &errorParse)
		if err != nil {
			return fmt.Errorf("found an empty parsed execution type for contract %s parsed as %s", c.CosmosMsgExecuteContract.Contract, MainnetCalcfiDCACodeID575FriendlyName)
		}

		var errorParseFields []string
		for key := range errorParse {
			errorParseFields = append(errorParseFields, key)
		}

		return fmt.Errorf("found an empty parsed execution type for contract %s parsed as %s; unparsed JSON has the following keys: %v", c.CosmosMsgExecuteContract.Contract, MainnetCalcfiDCACodeID575FriendlyName, errorParseFields)
	}

	switch {
	case executionType.ExecuteTrigger != nil:
		c.ExecutionType = DCA575ExecutionTypeExecuteTrigger
	default:
		vType := reflect.TypeOf(executionType)
		vValue := reflect.ValueOf(executionType)

		nonEmptyReflectTags := []string{}

		for i := 0; i < vType.NumField(); i++ {
			field := vType.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				continue
			}

			if !vValue.FieldByName(field.Name).IsZero() {
				nonEmptyReflectTags = append(nonEmptyReflectTags, jsonTag)
			}
		}
		c.ExecutionType = DCA575ExecutionType(fmt.Sprintf("unsupported fields: %v", nonEmptyReflectTags))
	}

	return nil
}

func IsEmpty(object interface{}) (bool, error) {
	if reflect.ValueOf(object).Kind() == reflect.Struct {
		empty := reflect.New(reflect.TypeOf(object)).Elem().Interface()
		if reflect.DeepEqual(object, empty) {
			return true, nil
		}
		return false, nil
	}
	return false, errors.New("check not implementend for this struct")
}

func (c *DCA575ParserHandler) ParseRelevantData() []parsingTypes.MessageRelevantInformation {
	return nil
}

func (c *DCA575ParserHandler) GetType() string {
	return wasm.MsgExecuteContract
}

func (c *DCA575ParserHandler) SetCosmosMsgExecuteContract(msg *wasmTypes.MsgExecuteContract) {
	c.CosmosMsgExecuteContract = msg
}
