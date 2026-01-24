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

### Part A: Server Setup (Do Once)

1.  **Systemd Service:**
    This ensures the blog runs in the background and restarts if it crashes.
    *   Copy [deploy/blog.service.example](./deploy/blog.service.example) to `/etc/systemd/system/blog.service` on your server.
    *   Edit it to set your username and paths.
    *   **Reload & Enable:**
        ```bash
        sudo systemctl daemon-reload
        sudo systemctl enable blog
        ```
    *(The service will be started automatically the first time you run the deploy script below).*

2.  **Passwordless Restart:**
    Allow the deployment script to restart the service without a password prompt.
    *   Run `sudo visudo` on the server.
    *   Add this line at the bottom (replace `youruser` with your actual username):
        ```text
        youruser ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart blog
        ```

3.  **Reverse Proxy (Caddy/Nginx):**
    Expose your app to the world with HTTPS.
    *   **Example Caddyfile:**
        ```text
        your-blog.com {
            reverse_proxy localhost:8080
        }
        ```

### Part B: Deploying Updates (Repeat Often)

Once the server is ready, you can deploy from your local machine with a single command.

1.  **Configure Local Settings:**
    Create `deploy/.env.deploy` with your server details (only needed once):
    ```bash
    cp deploy/.env.deploy.example deploy/.env.deploy
    ```

2.  **Run Deploy Script:**
    ```bash
    ./deploy/deploy.sh
    ```

This script will run your tests locally, build the Linux binary, sync the files (protecting your production DB), and restart the server automatically.

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
