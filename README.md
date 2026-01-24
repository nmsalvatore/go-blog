# Blog

A minimalist single-user blog.

## Development

```bash
cp .env.example .env
# Edit .env with your credentials
make run
```

Visit http://localhost:8080

## Deployment

### Prerequisites

- Linux server (Debian/Ubuntu)
- Domain name with DNS pointing to your server's IP address
- SSH access to your server
- Go 1.21+ installed locally (for compilation)

### 1. Build

Build for your server:

```bash
make build-linux
```

This cross-compiles a Linux binary (works from Mac/Windows). If building directly on your Linux server, use `make build` instead.

### 2. Upload

Copy files to your server. Replace `user@yourserver.com` with your actual SSH login:

```bash
# Create directory on server first
ssh user@yourserver.com "sudo mkdir -p /opt/blog"

# Copy files
scp blog user@yourserver.com:/tmp/
scp -r templates static .env.example user@yourserver.com:/tmp/

# Move to destination with correct ownership
ssh user@yourserver.com "sudo mv /tmp/blog /tmp/templates /tmp/static /tmp/.env.example /opt/blog/"
```

### 3. Create service user

Create a dedicated user to run the blog (more secure than running as root):

```bash
ssh user@yourserver.com
sudo useradd -r -s /bin/false blog
sudo chown -R blog:blog /opt/blog
```

### 4. Configure

Create and edit the environment file:

```bash
cd /opt/blog
sudo mv .env.example .env
sudo nano .env
```

Set your credentials:

```
ADMIN_USER=youruser
ADMIN_PASS=your-secure-password
SECURE_COOKIES=true
```

Secure the file (contains your password):

```bash
sudo chmod 600 .env
sudo chown blog:blog .env
```

### 5. Test

Verify the server starts:

```bash
cd /opt/blog
sudo -u blog ./blog
```

You should see `Server starting on :8080`. Press Ctrl+C to stop.

If it fails, check the error message. Common issues:
- Permission denied: check file ownership with `ls -la`
- Port in use: another service is using port 8080

### 6. Create systemd service

Create `/etc/systemd/system/blog.service`:

```bash
sudo nano /etc/systemd/system/blog.service
```

Paste this configuration:

```ini
[Unit]
Description=Blog
After=network.target

[Service]
Type=simple
User=blog
Group=blog
WorkingDirectory=/opt/blog
ExecStart=/opt/blog/blog
Restart=always
RestartSec=5
EnvironmentFile=/opt/blog/.env

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable blog
sudo systemctl start blog
```

Check it's running:

```bash
sudo systemctl status blog
```

If it fails, check the logs:

```bash
sudo journalctl -u blog -n 50
```

### 7. Install Caddy

```bash
sudo apt update
sudo apt install -y caddy
```

### 8. Configure Caddy

Edit the Caddyfile:

```bash
sudo nano /etc/caddy/Caddyfile
```

Replace the contents with (use your actual domain):

```
yourdomain.com {
    reverse_proxy localhost:8080
}
```

Restart Caddy:

```bash
sudo systemctl restart caddy
```

Caddy automatically obtains and renews TLS certificates from Let's Encrypt. This may take a moment on first start.

If certificates fail, check:
- DNS is pointing to this server (`dig yourdomain.com`)
- Ports 80 and 443 are open (`sudo ufw status`)

### 9. Configure firewall

Allow HTTP and HTTPS traffic:

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 10. Verify

Visit https://yourdomain.com - you should see your blog with HTTPS.

## Updating

To deploy changes:

```bash
# Local: build and upload
make build-linux
scp blog user@yourserver.com:/tmp/
scp -r templates static user@yourserver.com:/tmp/

# Server: move files and restart
ssh user@yourserver.com "sudo mv /tmp/blog /tmp/templates /tmp/static /opt/blog/ && sudo chown -R blog:blog /opt/blog && sudo systemctl restart blog"
```

Brief downtime (~1 second) occurs during restart.

## Backups

The database is a single file at `/opt/blog/blog.db`.

### Automatic local backups

Create a backup directory and cron job:

```bash
sudo mkdir -p /opt/blog/backups
sudo chown blog:blog /opt/blog/backups
```

Edit the crontab:

```bash
sudo crontab -e
```

Add this line to backup every 6 hours, keeping 7 days of backups:

```
0 */6 * * * cp /opt/blog/blog.db /opt/blog/backups/blog-$(date +\%Y\%m\%d-\%H).db && find /opt/blog/backups -name "blog-*.db" -mtime +7 -delete
```

### Restore from backup

```bash
sudo systemctl stop blog
sudo cp /opt/blog/backups/blog-YYYYMMDD-HH.db /opt/blog/blog.db
sudo chown blog:blog /opt/blog/blog.db
sudo systemctl start blog
```

### Offsite backups (recommended)

For important data, also copy backups offsite. Example using rsync to another server:

```bash
rsync -az /opt/blog/backups/ user@backupserver:/backups/blog/
```

## Troubleshooting

**Blog won't start:**
```bash
sudo journalctl -u blog -n 50
```

**Caddy won't get certificates:**
```bash
sudo journalctl -u caddy -n 50
# Verify DNS
dig yourdomain.com
```

**Permission errors:**
```bash
ls -la /opt/blog/
# Should show blog:blog ownership
sudo chown -R blog:blog /opt/blog
```

**Check if services are running:**
```bash
sudo systemctl status blog
sudo systemctl status caddy
```

### Updating

```bash
// change user to current user, pull changes, build 
sudo chown -R $USER:$USER .
git pull
make build

// change user back to blog
sudo chown -R blog:blog .
sudo systemctl restart blog
```
