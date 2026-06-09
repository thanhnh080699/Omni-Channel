# Workflow Command: /verify

Quy trình xác minh chất lượng và tính ổn định của mã nguồn sau khi chỉnh sửa.

---

## 📋 Mô Tả
Lệnh `/verify` được gọi sau khi hoàn thành việc viết code nhằm đảm bảo mã nguồn tuân thủ các quy tắc chất lượng, không bị lỗi cú pháp, vượt qua các bài kiểm thử và hoạt động trơn tru.

---

## 🔄 Các Bước Thực Hiện (Execution Steps)

### Bước 1: Kiểm Tra Tĩnh (Static Verification)
1.  Chạy các công cụ định dạng code (formatter) và linter của dự án để đảm bảo không có lỗi style:
    ```bash
    # Ví dụ
    npm run lint
    # hoặc
    flake8 .
    ```
2.  Chạy trình biên dịch hoặc kiểm tra kiểu tĩnh (typecheck):
    ```bash
    # Ví dụ
    npm run typecheck
    ```

### Bước 2: Chạy Kiểm Thử Tự Động (Automated Testing)
1.  Chạy toàn bộ test suite hoặc các test module liên quan trực tiếp đến tính năng vừa viết:
    ```bash
    # Ví dụ
    npm run test
    # hoặc
    pytest
    ```
2.  Đảm bảo tỷ lệ bao phủ kiểm thử (test coverage) đạt yêu cầu của dự án.

### Bước 3: Kiểm Tra Thực Tế (Runtime & Manual Verification)
1.  Chạy ứng dụng trong môi trường local dev server.
2.  Thực hiện gọi thử các API mới bằng curl hoặc kiểm tra trực tiếp giao diện trên trình duyệt để đảm bảo trải nghiệm người dùng (UX) và hiệu năng tải tốt.
