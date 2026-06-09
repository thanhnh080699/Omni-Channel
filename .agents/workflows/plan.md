# Workflow Command: /plan

Quy trình lập kế hoạch triển khai cho các tác vụ lập trình phức tạp.

---

## 📋 Mô Tả
Lệnh `/plan` được kích hoạt khi bắt đầu một tính năng mới, tái cấu trúc mã nguồn (refactoring) hoặc khi thực hiện một sửa đổi kiến trúc lớn. Mục tiêu là giúp AI Agent và người dùng đồng thuận về phương án kỹ thuật trước khi thay đổi bất kỳ dòng code nào.

---

## 🔄 Các Bước Thực Hiện (Execution Steps)

### Bước 1: Khảo Sát & Phân Tích (Research)
1.  Quét mã nguồn dự án để xác định:
    *   Các file sẽ bị ảnh hưởng trực tiếp.
    *   Các thư viện liên quan và phiên bản hiện tại.
    *   Các quy tắc lập trình tương ứng trong `.agents/rules/`.
2.  **Tuyệt đối không** tạo mới hay sửa đổi file code trong bước này.

### Bước 2: Tạo Bản Kế Hoạch (Design & Document)
1.  Tạo file `implementation_plan.md` tại thư mục `.system_generated/` hoặc thư mục do IDE chỉ định dựa trên biểu mẫu tại [templates/implementation_plan.template.md](file:///d:/work/ag-tool-kit/templates/implementation_plan.template.md).
2.  Ghi nhận đầy đủ:
    *   Mục tiêu kỹ thuật.
    *   Quyết định thiết kế và các thay đổi có thể gây lỗi hệ thống (breaking changes).
    *   Các câu hỏi chưa rõ ràng cần người dùng trả lời (Open Questions).
    *   Danh sách file sẽ sửa đổi được phân loại theo component.

### Bước 3: Phê Duyệt từ Người Dùng (User Approval)
1.  Gửi thông báo ngắn gọn cho người dùng biết kế hoạch đã sẵn sàng để review.
2.  **Dừng hoạt động** và chờ đợi sự phê duyệt hoặc phản hồi chỉnh sửa từ người dùng.
3.  Khi người dùng đã duyệt, tạo tiếp file `task.md` (Checklist công việc) để bắt đầu bước lập trình.
