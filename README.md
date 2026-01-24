# Go Blog Engine

👋 **Welcome!** This is a minimal, fast, and self-hosted blogging platform written in Go. It's designed for developers who want a simple, no-nonsense way to publish content without the bloat of WordPress or the complexity of static site generators.

It includes a built-in admin interface, Markdown support, drafts, and a "dummy-proof" deployment script.

## 🚀 Features

*   **Self-Contained:** Compiles to a single binary with an embedded SQLite database. No need to install Postgres, MySQL, or Redis.
*   **Minimal Markdown:** Support for **bold**, *italic*, and [links](https://example.com) only, keeping things clean and lightweight.
*   **Drafts:** Save posts as drafts and publish them when you're ready.
*   **RSS Feed:** Built-in automatic RSS feed at `/feed`.
*   **Secure:** CSRF protection, secure sessions, and strict HTML escaping.
*   **Themable:** Simple CSS variables for easy customization.

---

## 🛠️ Local Development

Getting started is super easy. You just need **Go 1.22+** installed.

1.  **Clone the repo:**
    ```bash
    git clone https://github.com/yourusername/go-blog.git
    cd go-blog
    ```

2.  **Setup Environment:**
    Copy the example configuration:
    ```bash
    cp .env.example .env
    ```
    *You can leave the defaults as-is for local testing.*

3.  **Run it:**
    We use a `Makefile` for convenience:
    ```bash
    make run
    ```
    Open your browser to [http://localhost:8080](http://localhost:8080).

4.  **Login:**
    *   Go to [http://localhost:8080/admin](http://localhost:8080/admin)
    *   **User:** `admin`
    *   **Password:** `changeme` (or whatever you set in `.env`)

---

## ⚙️ Configuration

Everything is configured via the `.env` file.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `ADMIN_USER` | Username for the admin panel. | `admin` |
| `ADMIN_PASS` | Password for the admin panel. | `changeme` |
| `SECURE_COOKIES` | Set to `true` in production (requires HTTPS). | `false` |
| `BLOG_NAME` | The name displayed in the header/title. | `My Blog` |

---

## 🚢 Deployment & Production

### 1. Reverse Proxy (Recommended)
In production, it is highly recommended to run this app behind a reverse proxy like **Caddy** or **Nginx**. This handles HTTPS (SSL/TLS) and provides a professional setup.

**Example Caddyfile:**
```text
your-blog.com {
    reverse_proxy localhost:8080
}
```

### 2. Systemd Service
You should run the blog as a systemd service to ensure it starts automatically on boot and restarts if it crashes.

See [deploy/blog.service.example](./deploy/blog.service.example) for a standard configuration template.

### 3. Passwordless Deployment
The deployment script restarts the `blog` service on your server. To make this "one-command" without typing a password every time, you can allow your user to restart the service via `sudo` without a password.

1. SSH into your server and run `sudo visudo`.
2. Add this line at the bottom (replace `youruser` with your actual username):
   ```text
   youruser ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart blog
   ```

### 4. One-Command Deploy
1. Configure your server details in `deploy/.env.deploy` (copy from `.env.deploy.example`).
2. Run the script:
   ```bash
   ./deploy/deploy.sh
   ```

The script will:
1. Run `go test` locally (and stop if they fail).
2. Build the binary for Linux.
3. Upload only the necessary files (it **won't** overwrite your production `blog.db` or `.env`).
4. Restart the service on your server instantly.

---

## 🤝 Contributing

We love contributions! Whether it's fixing a bug, improving the styles, or adding a new feature, feel free to fork the repo and submit a Pull Request.

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

---

**Happy Blogging!** ✍️
