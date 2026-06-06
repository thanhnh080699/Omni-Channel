# CDN Testing Suite

Bộ test case tích hợp cho dự án Meditour CDN, sử dụng `gin` test mode và `testify`.

## Cấu trúc
- `test_utils.go`: Các hàm bổ trợ để thiết lập router, mock config và dọn dẹp thư mục test.
- `health_test.go`: Kiểm tra trạng thái hệ thống và CORS.
- `folder_test.go`: Kiểm tra các tính năng tạo, đổi tên, xóa thư mục vật lý.
- `upload_test.go`: Kiểm tra tính năng upload (đơn lẻ), bao gồm cả kiểm tra nội dung file (Deep Inspection).
- `image_test.go`: Kiểm tra việc phục vụ ảnh và cơ chế xử lý/cache ảnh.
- `security_test.go`: Kiểm tra bảo mật API Key và Signed URLs.

## Cách chạy test
Chạy toàn bộ các test case:
```bash
go test -v ./tests/...
```

Chạy một file test cụ thể:
```bash
go test -v ./tests/folder_test.go ./tests/test_utils.go
```

## Lưu ý
Các bản test sẽ tạo ra thư mục tạm `test_uploads` trong quá trình chạy và tự động xóa sau khi hoàn tất. Thư mục này đã được đưa vào `.gitignore`.
