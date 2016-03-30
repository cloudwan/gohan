
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
create table `tests` (`description`text null,`id`varchar(255) primary key,`name`text null,`tenant_id`text null,`test5`text null);

create table `devices` (`management_ip`text null,`name`text null,`status`text null,`tenant_id`text null,`id`varchar(255) primary key);

create table `firewalls` (`tenant_id`text null,`id`varchar(255) primary key,`name`text not null);

create table `firewall_rules` (`tenant_id`text null,`firewall_id`varchar(255) not null,`action`text not null,`destination_port`text not null,`id`varchar(255) primary key,`protocol`text not null,foreign key(`firewall_id`) REFERENCES `firewalls`(id) ,foreign key(`firewall_id`) REFERENCES `firewalls`(id) );

create table `networks` (`firewall_id`varchar(255) null,`id`varchar(255) primary key,`name`text not null,`tenant_id`text null,foreign key(`firewall_id`) REFERENCES `firewalls`(id) );


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
drop table devices;

drop table firewalls;

drop table firewall_rules;

drop table networks;

drop table tests;

