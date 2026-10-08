#!/usr/bin/env bash
# Installs the KISY TURN relay (coturn) on a fresh Ubuntu 24.04 server with a
# public IPv4 address, e.g. a Hetzner Cloud instance. Run as root:
#
#   curl -fsSL https://raw.githubusercontent.com/hoffmann0027/kisy-project/main/deploy/coturn/install-ubuntu.sh | bash
#
# What it does:
#   - installs coturn and the firewall;
#   - writes /etc/turnserver.conf from deploy/coturn/turnserver.conf (the same
#     lockdown: no relaying into private, loopback, link-local or metadata
#     ranges, no TCP relay, quotas) plus this server's address and a freshly
#     generated shared secret;
#   - opens only SSH, TURN (3478 UDP/TCP) and the relay port range;
#   - prints the two values to set on the backend (TURN_URLS, TURN_SECRET).
#
# The secret is generated here and never leaves the server except on your
# screen: paste it into Render yourself. Running the script again keeps the
# existing secret, so the backend does not need updating.
set -euo pipefail

CONF_URL="https://raw.githubusercontent.com/hoffmann0027/kisy-project/main/deploy/coturn/turnserver.conf"
CONF=/etc/turnserver.conf
SECRET_FILE=/etc/kisy-turn-secret
MIN_PORT=49152
MAX_PORT=65535

if [ "$(id -u)" -ne 0 ]; then
  echo "Run as root." >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -yq coturn ufw curl openssl

# The address clients reach this server at. On Hetzner Cloud it sits on the
# interface itself, so the route lookup finds it; behind NAT set it by hand.
PUBLIC_IP="${PUBLIC_IP:-$(ip -4 route get 1.1.1.1 | awk '{for (i = 1; i < NF; i++) if ($i == "src") print $(i + 1)}')}"
if [ -z "$PUBLIC_IP" ]; then
  echo "Could not find this server's public IPv4; run again with PUBLIC_IP=<address>." >&2
  exit 1
fi

if [ ! -s "$SECRET_FILE" ]; then
  umask 077
  openssl rand -hex 32 > "$SECRET_FILE"
fi
SECRET="$(cat "$SECRET_FILE")"

# The repository's config, with this server's port range and identity.
curl -fsSL "$CONF_URL" \
  | sed -e "s/^min-port=.*/min-port=${MIN_PORT}/" -e "s/^max-port=.*/max-port=${MAX_PORT}/" \
  > "${CONF}.new"
cat >> "${CONF}.new" <<EOF

# --- this server (written by install-ubuntu.sh) ---
external-ip=${PUBLIC_IP}
realm=kisy
static-auth-secret=${SECRET}
syslog
EOF
install -m 640 -o root -g turnserver "${CONF}.new" "$CONF"
rm -f "${CONF}.new"

# Older packages start the daemon only when this is set.
if [ -f /etc/default/coturn ]; then
  sed -i 's/^#\?TURNSERVER_ENABLED=.*/TURNSERVER_ENABLED=1/' /etc/default/coturn
fi

ufw allow 22/tcp
ufw allow 3478/udp
ufw allow 3478/tcp
ufw allow "${MIN_PORT}:${MAX_PORT}/udp"
ufw --force enable

systemctl enable coturn
systemctl restart coturn
sleep 2
systemctl is-active --quiet coturn || {
  journalctl -u coturn -n 30 --no-pager
  echo "coturn did not start." >&2
  exit 1
}

cat <<EOF

coturn is running on ${PUBLIC_IP}.

Set these on the backend (Render -> kisy -> Environment), then save:

  TURN_URLS   = turn:${PUBLIC_IP}:3478?transport=udp,turn:${PUBLIC_IP}:3478?transport=tcp
  TURN_SECRET = ${SECRET}

Afterwards https://kisy.onrender.com/ready should report "turn":true.
EOF
