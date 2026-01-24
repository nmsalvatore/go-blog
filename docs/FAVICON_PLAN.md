# Plan: Custom Favicon Uploads

This document outlines the steps to implement a feature allowing administrators to upload a custom favicon for the blog.

## 1. Initial Setup & Defaults

- **Generate Default Favicons:** On application startup, check for the existence of `static/favicon.png` and `static/apple-touch-icon.png`. If they are missing, generate and save simple, default placeholder icons. This ensures that the links in `base.html` are always valid, even before a user has uploaded a custom icon.

## 2. Backend Implementation

### `image_processing.go`: Image Processing Logic

- Create a new file `image_processing.go` to house the image manipulation logic.
- Add helper functions:
  - `processFavicon(file io.Reader, outputPath string)`:
    1.  Decodes the uploaded image (supports JPEG, PNG, GIF).
    2.  Resizes the image to two standard sizes using `golang.org/x/image/draw`:
        - `32x32` for `favicon.png`.
        - `180x180` for `apple-touch-icon.png`.
    3.  Saves both resized images as PNGs to the `static/` directory.

### `handlers.go`: Settings Handler Update

- Modify the `Settings` handler to handle `multipart/form-data`.
- When a file is uploaded to the new `favicon` form field:
  1.  **Enforce strict upload limits:** Use `http.MaxBytesReader` to limit the total request size to 2MB.
  2.  **Validate the file:** Ensure it's a supported image type by checking its header.
  3.  **Handle Errors Gracefully:** If validation fails (e.g., file too large, not an image), re-render the settings page with a clear, user-friendly error message (e.g., "Invalid file type or size exceeds 2MB.").
  4.  Call `processFavicon` to handle resizing and saving.
  5.  Add a cache-busting query parameter to the favicon URL in the database/settings to ensure browsers refresh it. We can store the current timestamp as a `favicon_version` setting.

## 3. Frontend Changes

### `templates/settings.html`

- Add a new section to the settings form for "Favicon."
- Include an element to display any error messages related to the file upload.
- Include `<input type="file" name="favicon" accept="image/*">`.
- Display the current favicon next to the input for context.

### `templates/base.html`

- Update the `<head>` section to link to the new favicons.
- Use the `favicon_version` setting to add a cache-busting query string.
  ```html
  <link rel="icon" type="image/png" href="/static/favicon.png?v={{ .FaviconVersion }}">
  <link rel="apple-touch-icon" href="/static/apple-touch-icon.png?v={{ .FaviconVersion }}">
  ```

## 4. Deployment & Configuration

### `deploy/deploy.sh`

- Add `favicon.png` and `apple-touch-icon.png` to the `--exclude` list in the `rsync` command to prevent deployments from overwriting the user's uploaded icon.

### `.gitignore`

- Add `static/favicon.png` and `static/apple-touch-icon.png` to ensure these user-generated files are not accidentally committed to the repository.

## 5. Testing Strategy

### Unit Tests (`image_processing_test.go`)
- **Test Resizing:**
    - Create a dummy large image in memory.
    - Pass it to the processing function.
    - Assert that the output file exists, is a valid PNG, and has the correct dimensions (32x32 and 180x180).
- **Test Invalid Inputs:**
    - Verify that passing a non-image file or garbage data returns an appropriate error.

### Integration Tests (`handlers_test.go`)
- **Test `Settings` Upload:**
    - Use `multipart/form-data` to upload a test image to the settings endpoint.
    - Verify the handler returns `200 OK` or `303 See Other`.
    - Verify that the `favicon_version` setting is updated in the database.
- **Test Invalid Upload:**
    - Send a request with an invalid file (e.g., a text file) or a file that is too large.
    - Assert that the handler returns a `200 OK` status (re-rendering the page) and that the response body contains the expected error message.
    - *Note:* We will mock or use a temp directory for the file system writes to avoid cluttering the real `static/` folder during tests.

## 6. Dependencies

- Add `golang.org/x/image` to `go.mod` to support high-quality image resizing.
