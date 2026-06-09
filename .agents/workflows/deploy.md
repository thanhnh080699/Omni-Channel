# Workflow Command: /deploy

Quy trình đóng gói, ghi nhận thay đổi và sẵn sàng bàn giao sản phẩm.

---

## 📋 Mô Tả
Lệnh `/deploy` được gọi khi tất cả các tính năng đã được code xong và vượt qua bước xác minh chất lượng. Quy trình này tập trung vào việc tạo tài liệu bàn giao rõ ràng và hướng dẫn triển khai sản phẩm lên môi trường staging hoặc production.

---

## 🔄 Các Bước Thực Hiện (Execution Steps)

### Bước 1: Kiểm Tra Build Sản Phẩm (Production Build Validation)
1.  Chạy thử lệnh đóng gói dự án để chắc chắn bundle được build thành công, không gặp lỗi biên dịch trong môi trường production:
    ```bash
    # Ví dụ
    npm run build
    ```
2.  Kiểm tra kích thước các gói file sinh ra (bundle size) xem có vượt ngưỡng quy định không.

### Bước 2: Tạo Tài Liệu Bàn Giao (Walkthrough Documentation)
1.  Tạo hoặc cập nhật file `walkthrough.md` tại thư mục `.system_generated/` hoặc thư mục do IDE chỉ định dựa trên biểu mẫu tại [templates/walkthrough.template.md](file:///d:/work/ag-tool-kit/templates/walkthrough.template.md).
2.  Tài liệu phải liệt kê rõ:
    *   Các thay đổi đã thực hiện (kèm link clickable đến các file).
    *   Các kết quả kiểm thử (unit tests passed, lint passed).
    *   Đính kèm các hình ảnh, video minh họa (nếu có thay đổi về giao diện người dùng).

### Bước 3: Đẩy Code & Kích Hoạt CI/CD (Release & Deploy)
1.  Tạo commit tin nhắn đúng chuẩn Conventional Commits.
2.  Đẩy code lên nhánh remote (Git Push) để kích hoạt pipeline CI/CD (GitHub Actions, GitLab CI).
3.  Cung cấp link log của pipeline hoặc hướng dẫn deploy thủ công nếu dự án không cấu hình tự động.
