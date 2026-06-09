---
name: goravel-expert
description: Chuyên gia phát triển ứng dụng Go sử dụng Goravel Framework. Tối ưu hóa kiến trúc Service Container, Service Provider, ORM (GORM), Router và Middleware.
when_to_use: "Dự án phát hiện có file go.mod chứa 'github.com/goravel/framework' hoặc mã nguồn import package 'github.com/goravel/framework'"
---

# Kỹ Năng: Goravel Expert

Chỉ dẫn chuyên sâu này được tự động nạp khi phát hiện dự án sử dụng Goravel Framework (Framework web Go lấy cảm hứng từ Laravel).

---

## 🏗️ 1. Cấu Trúc MVC & Service Provider

*   **Service Container**: Hiểu cách đăng ký và giải quyết (resolve) các dependency thông qua Service Provider. Sử dụng `facades` để gọi các dịch vụ cốt lõi (như `facades.Orm()`, `facades.Config()`, `facades.Log()`).
*   **Tổ Chức Routing & Controller**:
    *   Khai báo Route trong `routes/web.go` hoặc `routes/api.go`.
    *   Controller chỉ nhận request thông qua `http.Context`, gọi service xử lý và trả về phản hồi qua `ctx.Response()`.

---

## 💾 2. ORM (Database) & Migrations

*   **Goravel ORM**: Sử dụng `facades.Orm().Query()` để thực hiện các câu lệnh database.
*   **Model Definition**: Định nghĩa model kế thừa struct `orm.Model` để có sẵn các trường `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`:
    ```go
    type User struct {
        orm.Model
        Name  string
        Email string
    }
    ```
*   **Migrations**: Luôn tạo và chạy database migration bằng CLI `go run . artisan make:migration [name]` thay vì sửa database trực tiếp.

---

## 🛡️ 3. Request Validation & Middleware

*   **Http Validation**: Sử dụng Validator của Goravel để kiểm tra payload request. Định nghĩa struct validation tags:
    ```go
    type RegisterRequest struct {
        Name  string `form:"name" json:"name" binding:"required"`
        Email string `form:"email" json:"email" binding:"required,email"`
    }
    ```
*   **Middleware**: Viết custom middleware để xử lý phân quyền, CORS, hoặc ghi log tùy biến cho từng nhóm route.
*   **Artisan Commands**: Sử dụng các lệnh Artisan tích hợp (`artisan make:controller`, `artisan make:model`) để khởi tạo các file mẫu nhanh chóng, đúng quy chuẩn.
