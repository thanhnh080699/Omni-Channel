---
name: golang-expert
description: Chuyên gia phát triển ứng dụng bằng ngôn ngữ Go (Golang). Định hình xử lý lỗi (error handling), đồng thời (concurrency), thiết kế Interface và tối ưu hóa hiệu năng.
when_to_use: "Dự án phát hiện có file go.mod hoặc các file mở rộng .go"
---

# Kỹ Năng: Go (Golang) Expert

Chỉ dẫn chuyên sâu này được tự động nạp khi phát hiện dự án sử dụng ngôn ngữ Go.

---

## 🏗️ 1. Cấu Trúc Dự Án & Package

*   **Standard Go Project Layout**: Tổ chức thư mục rõ ràng:
    *   `/cmd`: Chứa các entry point (hàm `main.go`) của ứng dụng.
    *   `/internal`: Chứa các package mã nguồn đóng, không cho phép import từ bên ngoài dự án.
    *   `/pkg`: Chứa các thư viện dùng chung có thể chia sẻ công khai.
*   **Tránh Import Vòng Lặp (Circular Dependency)**: Thiết kế các package độc lập, phân rã trách nhiệm rõ ràng. Nếu gặp lỗi import vòng, hãy sử dụng **Interface** ở package trung gian để giải quyết.

---

## 🛠️ 2. Xử Lý Lỗi (Error Handling)

*   **Error is Value**: Trong Go, lỗi là giá trị trả về tường minh. Luôn kiểm tra lỗi lập tức:
    ```go
    val, err := DoSomething()
    if err != nil {
        return fmt.Errorf("failed to do something: %w", err) // wrap error
    }
    ```
*   **Không Lạm Dụng Panic**: Chỉ sử dụng `panic` đối với các lỗi nghiêm trọng không thể phục hồi khi khởi chạy (ví dụ: không kết nối được Database chính). Đối với logic nghiệp vụ chạy runtime, bắt buộc phải trả về `error`.

---

## ⚡ 3. Đồng Thời (Concurrency) & Context

*   **Goroutine Leak**: Luôn đảm bảo goroutine được giải phóng khi không sử dụng. Tránh khởi tạo goroutine chạy vô hạn mà không có cơ chế dừng.
*   **Kênh truyền dẫn (Channels)**: Sử dụng channel để giao tiếp giữa các goroutine. Sử dụng `select` để xử lý timeout hoặc huỷ tác vụ.
*   **Context Propagation**: 
    *   Luôn truyền tham số `ctx context.Context` làm tham số đầu tiên của các hàm gọi I/O (Database, HTTP Request, File).
    *   Sử dụng `context.WithTimeout` hoặc `context.WithCancel` để tránh các truy vấn bị treo vô hạn.
