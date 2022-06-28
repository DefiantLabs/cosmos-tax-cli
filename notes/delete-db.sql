delete from tx_addresses;
delete from events;
delete from messages;
delete from txes;
delete from addresses;
delete from blocks;

-- Below will delete the table and the rows...

drop table if exists taxable_tx cascade;
drop table if exists chains cascade;
drop table if exists denom_unit_aliases cascade;
drop table if exists messages cascade;
drop table if exists txes cascade;
drop table if exists taxable_event cascade;
drop table if exists messages cascade;
drop table if exists addresses cascade;
drop table if exists blocks cascade;
drop table if exists simple_denoms cascade;
drop table if exists denom_units cascade;
drop table if exists denoms cascade;