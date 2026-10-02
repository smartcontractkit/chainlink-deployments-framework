package remote

import (
	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
)

func ThrowAndCatch(
	catalog *catalogDataStore,
	request *pb.DataAccessRequest,
) (*pb.DataAccessResponse, error) {
	return catalog.client.roundTrip(request)
}
