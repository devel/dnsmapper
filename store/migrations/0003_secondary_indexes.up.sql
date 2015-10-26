CREATE INDEX ips_server_asn ON ips (server_asn);
CREATE INDEX ips_server_cc ON ips (server_cc, server_rc);

CREATE INDEX ips_edns_asn ON ips (edns_asn);
CREATE INDEX ips_edns_cc  ON ips (edns_cc, edns_rc);

CREATE INDEX ips_client_asn ON ips (client_asn);
CREATE INDEX ips_client_cc  ON ips (client_cc, client_rc);
