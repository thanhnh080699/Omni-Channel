---
name: php-laravel-expert
description: Chuyên gia phát triển ứng dụng web PHP sử dụng Laravel Framework. Thiết kế Service Container, Eloquent ORM, Form Request Validation, API Resources, Queues và Artisan Commands.
when_to_use: "Dự án phát hiện có file composer.json chứa 'laravel/framework' hoặc file 'artisan' ở root, hoặc các file mở rộng .php"
---

# Kỹ Năng: PHP & Laravel Expert

Chỉ dẫn chuyên sâu này được tự động nạp khi phát hiện dự án sử dụng PHP và Laravel Framework.

---

## 🏗️ 1. Service Container & IoC

*   **Dependency Injection**: Tận dụng tối đa Service Container để tự động inject các class dependencies qua `constructor` hoặc method injection.
*   **Service Providers**: Đăng ký các binding, singleton, hoặc boot-up logic trong các Service Provider (ví dụ: `AppServiceProvider`).
*   **Facades vs Contracts**: Ưu tiên sử dụng Dependency Injection qua Contracts để tăng khả năng viết unit test dễ dàng. Chỉ sử dụng Facades cho các tác vụ nhanh hoặc script nhỏ.

---

## 💾 2. Eloquent ORM & Migrations

*   **Eloquent Relationships**: Khai báo rõ ràng các quan hệ (`hasMany`, `belongsTo`, `belongsToMany`) và luôn sử dụng **Eager Loading** (`with()`) để phòng chống lỗi truy vấn N+1.
*   **Database Migrations**:
    *   Mọi thay đổi database schema bắt buộc phải được thực hiện qua file Migration.
    *   Đảm bảo hàm `down()` được định nghĩa chính xác để hỗ trợ rollback hoàn toàn.
*   **Mass Assignment Protection**: Sử dụng `$fillable` hoặc `$guarded` trong Model để bảo vệ hệ thống khỏi tấn công gán dữ liệu hàng loạt.

---

## 🛡️ 3. HTTP Layer & Validation

*   **Form Requests**: Không viết validation logic trong Controller. Luôn tạo các class Form Request riêng biệt (`php artisan make:request`) để xử lý phân quyền và kiểm tra tính hợp lệ của payload.
*   **API Resources**: Sử dụng `Eloquent Resources` (`php artisan make:resource`) để định dạng JSON đầu ra, tách biệt cấu trúc bảng database vật lý với API public.
*   **Middleware**: Áp dụng các middleware bảo mật mặc định (xác thực CSRF, Auth, Rate Limiting).
*   **Artisan Commands**: Luôn sử dụng lệnh Artisan để sinh mã nguồn chuẩn hóa cấu trúc thư mục của Laravel.
*   **Queues & Jobs**: Đưa các tác vụ nặng (như gửi email, xử lý ảnh, đồng bộ dữ liệu bên thứ ba) vào Queue Background Job để tránh làm treo request của người dùng.
