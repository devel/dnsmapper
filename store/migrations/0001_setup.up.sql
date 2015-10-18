create table IF NOT EXISTS ips (
    client_ip inet not null,
    server_ip inet not null,
    edns_net  cidr null,
    client_cc char(2) null,
    client_rc char(2) null,
    server_cc char(2) null,
    server_rc char(2) null,
    edns_cc char(2) null,
    edns_rc char(2) null,
    client_asn int null,
    server_asn int null,
    edns_asn int null,
    has_edns boolean,
    test_ip  inet,
    first_seen timestamp with time zone,
    last_seen timestamp with time zone
);

create index ips_client_idx if not exists on ips (client_ip, server_ip);
