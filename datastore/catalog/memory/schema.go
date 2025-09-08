package memory

const (
	sCHEMA_ADDRESS_REFERENCES = `
		CREATE TABLE address_references (
		chain_selector           bigint not null,
		contract_type            varchar(255) not null,
		version                  varchar(255) not null,
		qualifier                varchar(255) not null,
		address                  varchar(255) not null,
		label_set				 text[],

		PRIMARY KEY(chain_selector, contract_type, version, qualifier)
		);`

	sCHEMA_CONTRACT_METADATA = `
		CREATE TABLE contract_metadata (
			chain_selector           bigint not null,
			address                  varchar(255) not null,
			metadata                 text,

			PRIMARY KEY(chain_selector, address)
		);`

	sCHEMA_CHAIN_METADATA = `
		CREATE TABLE chain_metadata (
			chain_selector           bigint not null,
			metadata                 text,

			PRIMARY KEY(chain_selector)
		);`

	sCHEMA_ENVIRONMENT_METADATA = `
		CREATE TABLE environment_metadata (
			id        INTEGER not null,
			metadata  text,

			PRIMARY KEY(id)
		);`
)
