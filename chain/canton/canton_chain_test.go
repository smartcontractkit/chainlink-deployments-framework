package canton

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestChain_ChainInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   uint64
		wantName   string
		wantString string
		wantFamily string
	}{
		{
			name:       "returns correct info",
			selector:   chainsel.CANTON_TESTNET.Selector,
			wantName:   chainsel.CANTON_TESTNET.Name,
			wantString: "canton-testnet (9268731218649498074)",
			wantFamily: chainsel.FamilyCanton,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c := Chain{
				ChainMetadata: ChainMetadata{Selector: test.selector},
			}

			assert.Equal(t, test.selector, c.ChainSelector())
			assert.Equal(t, test.wantString, c.String())
			assert.Equal(t, test.wantName, c.Name())
			assert.Equal(t, test.wantFamily, c.Family())
		})
	}
}

func TestCreateLedgerServiceClients(t *testing.T) {
	var conn grpc.ClientConnInterface
	ledgerServiceClients := CreateLedgerServiceClients(conn)
	assertNoFieldIsZero(t, ledgerServiceClients)
	assertNoFieldIsZero(t, ledgerServiceClients.Admin)
}

func TestCreateAdminServiceClients(t *testing.T) {
	var conn grpc.ClientConnInterface
	adminServiceClients := CreateAdminServiceClients(conn)
	assertNoFieldIsZero(t, adminServiceClients)
}

// assertNoFieldIsZero checks that all fields of a struct are non-zero. If any field is zero, it fails the test and reports which fields were zero.
func assertNoFieldIsZero(t *testing.T, structValue any, msgAndArgs ...any) {
	t.Helper()

	var emptyFields []string
	structT := reflect.TypeOf(structValue)
	structV := reflect.ValueOf(structValue)
	for i := 0; i < structT.NumField(); i++ {
		field := structT.Field(i)
		if structV.Field(i).IsZero() {
			emptyFields = append(emptyFields, field.Name)
		}
	}

	if len(emptyFields) > 0 {
		assert.Fail(t, fmt.Sprintf("Expected all fields to be set, but the following fields were zero: %s", strings.Join(emptyFields, ", ")), msgAndArgs...)
	}
}
