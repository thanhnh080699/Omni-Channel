---
type: project
created: 2026-06-08
updated: 2026-06-08
---

# Project Conventions (Quy Ước Dự Án)

Tài liệu này ghi lại các quy ước chung về quy trình phát triển và coding style của dự án.

---

## 1. Quy Trình Git Workflow

*   **Tạo Nhánh (Branching)**: Mọi thay đổi lớn hoặc tính năng mới bắt buộc phải tạo nhánh riêng từ nhánh `main` (hoặc `develop`).
    *   *Định dạng tên nhánh*: `<type>/<issue-id>-<short-description>` (Ví dụ: `feat/102-jwt-auth`, `fix/99-cors-error`).
*   **Pull Requests (PR)**:
    *   Đảm bảo code đã chạy qua script linter và unit test local thành công trước khi đẩy code lên remote.
    *   Mỗi PR cần có ít nhất một reviewer phê duyệt trước khi merge vào nhánh chính.