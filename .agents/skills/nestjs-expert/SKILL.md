---
name: nestjs-expert
description: Chuyên gia phát triển ứng dụng backend sử dụng NestJS (Node.js/TypeScript). Định hình thiết kế Module, Controller, Provider, DTO, Pipes, Guards và Exception Filters.
when_to_use: "Dự án phát hiện có package.json chứa '@nestjs/core' hoặc các file kết thúc bằng .controller.ts, .module.ts, .service.ts"
---

# Kỹ Năng: NestJS Expert

Chỉ dẫn chuyên sâu này được tự động nạp khi phát hiện dự án sử dụng NestJS Framework.

---

## 🏗️ 1. Kiến Trúc Cốt Lõi (Core Architecture)

*   **Tính Đóng Gói (Encapsulation)**: Mọi chức năng phải được đóng gói gọn gàng trong các Module. Tránh khai báo import chéo không có định hướng, luôn sử dụng `@Module` để xuất (exports) các Service cần dùng chung.
*   **Dependency Injection (DI)**: Sử dụng decorator `@Injectable()` cho các Provider/Service và inject chúng thông qua `constructor` (Constructor Injection).
*   **Tách Biệt Nhiệm Vụ (Separation of Concerns)**:
    *   **Controller**: Chỉ làm nhiệm vụ nhận request, gọi Service và trả về response. Không viết business logic ở Controller.
    *   **Service/Provider**: Nơi chứa toàn bộ logic xử lý nghiệp vụ, giao tiếp cơ sở dữ liệu.

---

## 🛡️ 2. Validation & Security

*   **DTO (Data Transfer Object)**: Luôn khai báo class DTO cho các request payload (`Body`, `Query`, `Param`).
*   **class-validator & class-transformer**: Sử dụng decorator (như `@IsString()`, `@IsNotEmpty()`, `@IsEmail()`, `@IsOptional()`) để ràng buộc kiểu dữ liệu trực tiếp trong DTO.
*   **ValidationPipe**: Đảm bảo dự án kích hoạt `ValidationPipe` toàn cục với tùy chọn `whitelist: true` để tự động loại bỏ các thuộc tính không được định nghĩa trong DTO:
    ```typescript
    app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
    ```

---

## 🚀 3. Tối Ưu Hóa & Lọc Lỗi

*   **Exception Filters**: Sử dụng HttpExceptions tiêu chuẩn của NestJS (`NotFoundException`, `BadRequestException`, `ForbiddenException`) thay vì ném lỗi chuỗi thô. Viết Custom Exception Filter để chuẩn hóa JSON lỗi trả về.
*   **Guards & Interceptors**:
    *   Sử dụng **Guards** cho các tác vụ phân quyền và xác thực (AuthGuard, RolesGuard).
    *   Sử dụng **Interceptors** để định dạng lại dữ liệu trả về (Response Transform) hoặc ghi log thời gian xử lý request.
