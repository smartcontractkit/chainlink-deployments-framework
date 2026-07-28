package analyzer

import (
	mcmssuisdk "github.com/smartcontractkit/mcms/sdk/sui"
)

const (
	suiExecutionPackageIDFieldName        = "execution_package_id"
	suiExecutionPackageIDsFieldName       = "execution_package_ids"
	suiExecutionContractTypeFieldName     = "execution_contract_type"
	suiExecutionContractVersionFieldName  = "execution_contract_version"
)

// suiPackageRoutingInputs surfaces upgraded-package dispatch metadata that is stored
// separately from tx.To. tx.To remains the MCMS on-chain identity (genesis package);
// latest_package_id and internal_latest_package_ids route the actual MoveCall target.
func suiPackageRoutingInputs(
	ctx ProposalContext,
	chainSelector uint64,
	identityPackageID string,
	fields mcmssuisdk.AdditionalFields,
) []NamedField {
	var routing []NamedField

	if executionPackageID := fields.LatestPackageID; executionPackageID != "" && executionPackageID != identityPackageID {
		routing = append(routing, suiExecutionPackageField(executionPackageID)...)
		routing = append(routing, suiExecutionContractInfoFields(ctx, chainSelector, executionPackageID)...)
	}

	if len(fields.InternalLatestPackageIDs) > 0 {
		elements := make([]FieldValue, 0, len(fields.InternalLatestPackageIDs))
		for _, packageID := range fields.InternalLatestPackageIDs {
			if packageID == "" {
				elements = append(elements, SimpleField{Value: ""})
				continue
			}
			elements = append(elements, AddressField{Value: packageID})
		}
		routing = append(routing, NamedField{
			Name:  suiExecutionPackageIDsFieldName,
			Value: ArrayField{Elements: elements},
		})
	}

	return routing
}

func suiExecutionPackageField(executionPackageID string) []NamedField {
	return []NamedField{{
		Name:  suiExecutionPackageIDFieldName,
		Value: AddressField{Value: executionPackageID},
	}}
}

func suiExecutionContractInfoFields(ctx ProposalContext, chainSelector uint64, executionPackageID string) []NamedField {
	contractType, contractVersion := lookupContractInfoByAddress(ctx, chainSelector, executionPackageID, "", "")
	if contractType == "" && contractVersion == "" {
		return nil
	}

	var fields []NamedField
	if contractType != "" {
		fields = append(fields, NamedField{
			Name:  suiExecutionContractTypeFieldName,
			Value: SimpleField{Value: contractType},
		})
	}
	if contractVersion != "" {
		fields = append(fields, NamedField{
			Name:  suiExecutionContractVersionFieldName,
			Value: SimpleField{Value: contractVersion},
		})
	}

	return fields
}

func prependSuiInputs(routing, inputs []NamedField) []NamedField {
	if len(routing) == 0 {
		return inputs
	}
	if len(inputs) == 0 {
		return routing
	}

	out := make([]NamedField, 0, len(routing)+len(inputs))
	out = append(out, routing...)
	out = append(out, inputs...)

	return out
}
