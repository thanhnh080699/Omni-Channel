# Meditour CDN Service (Go)

Dịch vụ CDN độc lập hiệu năng cao được xây dựng bằng Go và Gin để xử lý việc tải lên tệp (upload) và phục vụ các tệp tĩnh (static files).

## Các Tính Năng Chính

-   **Lưu Trữ**: Dựa trên lưu trữ cục bộ (Local Storage) với cấu trúc thư mục được sắp xếp khoa học (`YYYY/MM/DD/uuid.ext`).
-   **Quản Lý Thư Mục**: Endpoint tạo và xóa thư mục vật lý chủ động.
-   **Xử lý Ảnh Tức Thì (On-the-fly Processing)**: Hỗ trợ thay đổi kích thước (`w`, `h`) và định dạng (`fmt=webp/jpg/png`) qua tham số URL cho các loại tệp hình ảnh.
-   **Cơ Chế Cache**: Tự động lưu lại các phiên bản ảnh đã xử lý vào thư mục cache để tối ưu hiệu năng cho các lượt truy cập sau.
-   **Kiểm Tra Tệp Sâu (Deep Inspection)**: Đọc nội dung tệp (Magic Bytes) để xác thực định dạng thực tế, ngăn chặn việc giả mạo đuôi file hoặc tải lên mã độc.
-   **Signed URLs (Chữ ký URL)**: Cơ chế bảo mật truy cập tệp bằng chữ ký HMAC, ngăn chặn việc truy cập trái phép hoặc dò tìm file (có thể bật/tắt qua config).
-   **Rate Limiting**: Giới hạn số lượng request trên mỗi IP để chống DDOS/Brute-force.
-   **Kiểm Thử (Testing)**: Bộ test case tích hợp (Integration Tests) cho tất cả tính năng chính.
-   **Phục Vụ Tệp**: Máy chủ tệp tĩnh với header `Cache-Control` được tối ưu cho trình duyệt.
-   **Bảo Mật**: Xác thực cơ bản qua API Key thông qua header hoặc tham số truy vấn (query parameter).
-   **CORS**: Hỗ trợ Cross-Origin Resource Sharing, có thể cấu hình các domain được phép truy cập.
-   **Định Dạng Chuẩn**: Trả về phản hồi JSON bao gồm URL công khai, đường dẫn tệp, kích thước và siêu dữ liệu.
-   **Tùy Chỉnh**: Có thể cấu hình kích thước tải lên tối đa và các định dạng tệp được phép.

## Cấu Trúc Thư Mục

## Cài Đặt

1.  **Di chuyển vào thư mục dự án**:
    ```bash
    cd cdn
    ```

2.  **Cấu Hình**:
    Sao chép hoặc chỉnh sửa tệp `.env`:
    ```env
    PORT=8081
    UPLOAD_DIR=./uploads
    ALLOWED_EXTENSIONS=jpg,jpeg,png,gif,webp,svg,mp4,mov,avi,webm,pdf,doc,docx,xls,xlsx,ppt,pptx,txt,zip,rar
    MAX_UPLOAD_SIZE=104857600 # 100MB
    API_KEY=meditour_cdn_secret_key
    BASE_URL=http://localhost:8081
    ```

3.  **Cài Đặt Các Phụ Thuộc (Dependencies)**:
    ```bash
    go mod tidy
    ```

4.  **Chạy Dịch Vụ**:
    ```bash
    go run main.go
    ```

## Tài Liệu API

### 1. Kiểm Tra Trạng Thái (Health Check)
-   **Endpoint**: `GET /health`
-   **Phản hồi**: `{"status": "UP", "info": "Meditour CDN Service"}`

### 2. Tải Lên Một Tệp (Single File Upload)
-   **Endpoint**: `POST /api/upload`
-   **Xác thực**: Cần header `X-API-KEY`.
-   **Request Body**: `multipart/form-data`
    -   `file`: Tệp cần tải lên (bắt buộc).
    -   `folder`: Thư mục đích (tùy chọn). Ví dụ: `products/2026/electronics`.
-   **Ví dụ Phản Hồi**:
    ```json
    {
        "message": "File uploaded successfully",
        "data": {
            "original_name": "image.jpg",
            "file_name": "3f4b5c...jpg",
            "path": "2026/03/06/3f4b5c...jpg",
            "url": "http://localhost:8081/uploads/2026/03/06/3f4b5c...jpg",
            "size": 102400,
            "mime_type": "image/jpeg"
        }
    }
    ```

### 3. Tải Lên Nhiều Tệp (Multiple File Upload)
-   **Endpoint**: `POST /api/uploads`
-   **Auth**: `X-API-KEY` header required.
-   **Request Body**: `multipart/form-data`
    -   `files`: Các tệp cần tải lên (nhiều trường).
    -   `folder`: Thư mục đích (tùy chọn).

### 4. Quản Lý Thư Mục (Folder Management)
-   **Tạo Thư Mục**:
    -   **Endpoint**: `POST /api/folder`
    -   **Auth**: `X-API-KEY` header required.
    -   **Body**: `{"path": "products/2026/electronics"}`
-   **Đổi Tên Thư Mục**:
    -   **Endpoint**: `PUT /api/folder`
    -   **Auth**: `X-API-KEY` header required.
    -   **Body**: `{"old_path": "products/2026/old", "new_path": "products/2026/new"}`
-   **Xóa Thư Mục**:
    -   **Endpoint**: `DELETE /api/folder`
    -   **Auth**: `X-API-KEY` header required.
    -   **Body**: `{"path": "products/2026/electronics"}`
    -   **Lưu ý**: Hành động này sẽ xóa toàn bộ file và thư mục con một cách đệ quy (tương đương `rm -rf`).

### 5. Quản Lý Tệp (File Management)
-   **Xóa Tệp**:
    -   **Endpoint**: `DELETE /api/file`
    -   **Auth**: `X-API-KEY` header required.
    -   **Query Param**: `path=products/2026/image.jpg`
    -   **Lưu ý**: Hành động này sẽ xóa tệp gốc và **toàn bộ các biến thể cache** của tệp đó (resize, webp, v.v.).

### 6. Truy Cập Tệp
-   **Endpoint**: `GET /uploads/{path}`
-   **Tham số mẫu**:
    -   `?w=300`: Chiều rộng 300px (tự động tính chiều cao để giữ tỉ lệ).
    -   `?h=200`: Chiều cao 200px (tự động tính chiều rộng).
    -   `?w=300&h=200`: Cắt/Resize về đúng 300x200px.
    -   `?fmt=webp`: Tự động chuyển đổi định dạng ảnh gốc sang `webp`.
    -   `?sig=...&exp=...`: Chữ ký bảo mật (nếu `REQUIRE_SIGNATURE=true`).
-   **Phản hồi**: Trả về dữ liệu binary của ảnh. Các file đã xử lý sẽ được lưu vào thư mục `CACHE_DIR` để tăng tốc cho các yêu cầu sau.
-   **Cách tính toán signature (Golang)**: `HMAC_SHA256(path + expiry, signature_key)`

### 6. Chạy Kiểm Thử (Testing)
Dự án đi kèm với bộ test case tích hợp để đảm bảo tính ổn định của các tính năng (Upload, Folder, Security, Processing).

Để chạy toàn bộ các bài kiểm thử:
```bash
go test -v ./tests/...
```

## Bảo Mật & Lưu Ý
-   **API Key**: Luôn sử dụng header `X-API-KEY` khi tương tác với API.
-   **Signed URLs**: Khi được kích hoạt, bạn phải cung cấp thêm tham số `sig` và `exp` để truy cập ảnh.
-   **Rate Limit**: Giới hạn mặc định có thể điều chỉnh trong `.env`.
