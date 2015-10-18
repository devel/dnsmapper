DELETE FROM ips AS P1
USING ips AS P2
WHERE (P1.last_seen > P2.last_seen
       OR (P1.last_seen = P2.last_seen
           AND P1.first_seen > P2.first_seen))
   AND P1.server_ip = P2.server_ip
   AND P1.client_ip = P2.client_ip;

CREATE UNIQUE INDEX ips_ip_uidx ON ips (server_ip, client_ip);

