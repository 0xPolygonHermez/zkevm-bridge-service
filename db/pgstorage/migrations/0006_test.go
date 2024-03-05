package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This migration creates a different proof table dropping all the information.

type migrationTest0006 struct{}

func (m migrationTest0006) InsertData(db *sql.DB) error {
	block := "INSERT INTO sync.block (id, block_num, block_hash, parent_hash, network_id, received_at) VALUES(609636, 2803824, decode('27474F16174BBE50C294FE13C190B92E42B2368A6D4AEB8A4A015F52816296C3','hex'), decode('C9B5033799ADF3739383A0489EFBE8A0D4D5E4478778A4F4304562FD51AE4C07','hex'), 1, '0001-01-01 01:00:00.000');"
	if _, err := db.Exec(block); err != nil {
		return err
	}
	insert := "INSERT INTO sync.monitored_txs (id, block_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at) VALUES(130161, 609636, decode('34353AC3B4EB2F4DE26845ECE44733453F74CAAF','hex'), decode('F6BEEEBB578E214CA9E23B0E9683454FF88ED2A7','hex'), 45155, '<nil>', decode('2CFFD02E2C42C143213FD0E36D843D9D40866CE7BE02C671BEEC0EAE3FFD3D2638ACC87CAD3228B676F7D3CD4284A5443F17F1962B36E491B30A40B2405849E597BA5FB5B4C11951957C6F8F642C4AF61CD6B24640FEC6DC7FC607EE8206A99E92410D3021DDB9A356815C3FAC1026B6DEC5DF3124AFBADB485C9BA5A3E3398A04B7BA85560772B348ED365DB06F0733574CD1EEB40C499F589E6CB697F4C5013EFDF41131BB4E597286B6408422BC670ACCAE918EFFC64D836DFD7659A090C54A4CB8E8E198CC9405CDC50C64C07BD4C21BBD3B717540B448AA9CA6DE1CDD8A3A475CD6FFD70157E48063FC33C97A050F7F640233BF646CC98D9524C6B92BCF3AB56F839867CC5F7F196B93BAE1E27E6320742445D290F2263827498B54FEC539F756AFCEFAD4E508C098B9A7E1D8FEB19955FB02BA9675585078710969D3440F5054E0B36BE68D55E47EF092DD266AC32C93CD026408B38F973E7E2B878399AA36F64A729B2A4D1D8A59F8EF0337F976980B3A71AE487BB7AAD320B74598DE3E8359B1B12B53E2846D04411D8E3FAF24C5831C3FFE8BC07B2E620B1791A49D2C5DCEDAC7478ACEB09309F31467A671AAF20E095E63F7F3DDBFAC998A572588A35C95819F833381E59F86473AB1AC4BE6ECCE76C9FDBC30161665D8DFB120B344C968ED845E4D97C17BFFFE24471BE7DD959E3F12AD65BE053FB3D574F1E640269C13D8645B9978AD76C57DB977FE48EFB0B43110E3EBE78F9BADD9047AAC6BCD8D66A5E1D3B5C807B281E4683CC6D6315CF95B9ADE8641DEFCB32372F1C126E398EF7A5A2DCE0A8A7F68BB74560F8F71837C2C2EBBCBF7FFFB42AE1896F13F7C7479A0B46A28B6F55540F89444F63DE0378E3D121BE09E06CC9DED1C20E65876D36AA0C65E9645644786B620E2DD2AD648DDFCBF4A7E5B1A3A4ECFE7F64667A3F0B7E2F4418588ED35A2458CFFEB39B93D26F18D2AB13BDCE6AEE58E7B99359EC2DFD95A9C16DC00D6EF18B7933A6F8DC65CCB55667138776F7DEA101070DC8796E3774DF84F40AE0C8229D0D6069E5C8F39A7C299677A09D367FC7B05E3BC380EE652CDC72595F74C7B1043D0E1FFBAB734648C838DFB0527D971B602BC216C9619EF0ABF5AC974A1ED57F4050AA510DD9C74F508277B39D7973BB2DFCCC5EEB0618DB8CD74046FF337F0A7BF2C8E03E10F642C1886798D71806AB1E888D9E5EE87D0838C5655CB21C6CB83313B5A631175DFF4963772CCE9108188B34AC87C81C41E662EE4DD2DD7B2BC707961B1E646C4047669DCB6584F0D8D770DAF5D7E7DEB2E388AB20E2573D171A88108E79D820E98F26C0B84AA8B2F4AA4968DBB818EA32293237C50BA75EE485F4C22ADF2F741400BDF8D6A9CC7DF7ECAE576221665D7358448818BB4AE4562849E949E17AC16E0BE16688E156B5CF15E098C627C0056A9000000000000000000000000000000000000000000000000000000000001FC714C39479BB3A50DAC1EF581C50DFDCBAC84C5DEF27CFC2F4E311DAB8E365EA780B59D3B536A4C3F03F4892F84474FE2851E42BC60B8A47807702BCFEF7CECC7A90000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000009AF3049DD15616FD627A35563B5282BEA5C32E2000000000000000000000000000000000000000000000000000005AF3107A400000000000000000000000000000000000000000000000000000000000000005200000000000000000000000000000000000000000000000000000000000000000','hex'), 101786, 'confirmed', '{decode(''5C7838373762353766343562386466626462653832396666646562343164626333333938303835653333626461343664613338613332393738323333316562303061'',''hex'')}', '2023-10-03 10:29:08.283', '2023-10-03 10:29:09.491');"
	if _, err := db.Exec(insert); err != nil {
		return err
	}
	return nil
}

func (m migrationTest0006) RunAssertsAfterMigrationUp(t *testing.T, db *sql.DB) {
	// Check Primary Keys
	tokenPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'sync' and table_name = 'token_wrapped' and constraint_type = 'PRIMARY KEY';"
	row := db.QueryRow(tokenPK)
	var count int
	assert.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)

	rhtPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'rht' and constraint_type = 'PRIMARY KEY';"
	row2 := db.QueryRow(rhtPK)
	var count2 int
	assert.NoError(t, row2.Scan(&count2))
	assert.Equal(t, 1, count2)

	rootPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'root' and constraint_type = 'PRIMARY KEY';"
	row3 := db.QueryRow(rootPK)
	var count3 int
	assert.NoError(t, row3.Scan(&count3))
	assert.Equal(t, 1, count3)

	rhtTemp := "select count(*) from mt.rht_temp;"
	row4 := db.QueryRow(rhtTemp)
	var count4 int
	assert.Error(t, row4.Scan(&count4))

	indexes := []string{"claim_block_id", "deposit_block_id", "token_wrapped_block_id"}
	// Check indexes adding
	for _, idx := range indexes {
		// getIndex
		const getIndex = `SELECT count(*) FROM pg_indexes WHERE indexname = $1;`
		row := db.QueryRow(getIndex, idx)
		var result int
		assert.NoError(t, row.Scan(&result))
		assert.Equal(t, 1, result)
	}

	checkBlockID := "select block_id from sync.monitored_txs;"
	row5 := db.QueryRow(checkBlockID)
	var blockID int
	assert.Error(t, row5.Scan(&blockID))

	checkDepositID := "select deposit_id from sync.monitored_txs;"
	row6 := db.QueryRow(checkDepositID)
	var depositID int
	assert.NoError(t, row6.Scan(&depositID))
	assert.Equal(t, 130161, depositID)
}

func (m migrationTest0006) RunAssertsAfterMigrationDown(t *testing.T, db *sql.DB) {
	// Check Primary Keys
	tokenPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'sync' and table_name = 'token_wrapped' and constraint_type = 'PRIMARY KEY';"
	row := db.QueryRow(tokenPK)
	var count int
	assert.NoError(t, row.Scan(&count))
	assert.Equal(t, 0, count)

	rhtPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'rht' and constraint_type = 'PRIMARY KEY';"
	row2 := db.QueryRow(rhtPK)
	var count2 int
	assert.NoError(t, row2.Scan(&count2))
	assert.Equal(t, 0, count2)

	rootPK := "select count(constraint_name) from information_schema.table_constraints where table_schema = 'mt' and table_name = 'root' and constraint_type = 'PRIMARY KEY';"
	row3 := db.QueryRow(rootPK)
	var count3 int
	assert.NoError(t, row3.Scan(&count3))
	assert.Equal(t, 0, count3)

	rhtTemp := "select count(*) from mt.rht_temp;"
	row4 := db.QueryRow(rhtTemp)
	var count4 int
	assert.NoError(t, row4.Scan(&count4))
	assert.Equal(t, 0, count4)

	indexes := []string{"claim_block_id", "deposit_block_id", "token_wrapped_block_id"}
	// Check indexes removing
	for _, idx := range indexes {
		// getIndex
		const getIndex = `SELECT count(*) FROM pg_indexes WHERE indexname = $1;`
		row := db.QueryRow(getIndex, idx)
		var result int
		assert.NoError(t, row.Scan(&result))
		assert.Equal(t, 0, result)
	}

	checkBlockID := "select block_id from sync.monitored_txs;"
	row5 := db.QueryRow(checkBlockID)
	var blockID int
	assert.NoError(t, row5.Scan(&blockID))
	assert.Equal(t, 0, blockID)

	checkDepositID := "select deposit_id from sync.monitored_txs;"
	row6 := db.QueryRow(checkDepositID)
	var depositID int
	assert.Error(t, row6.Scan(&depositID))

	checkID := "select id from sync.monitored_txs;"
	row7 := db.QueryRow(checkID)
	var ID int
	assert.NoError(t, row7.Scan(&ID))
	assert.Equal(t, 130161, ID)
}

func TestMigration0006(t *testing.T) {
	runMigrationTest(t, 6, migrationTest0006{})
}
