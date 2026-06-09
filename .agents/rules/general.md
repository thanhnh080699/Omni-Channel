---
trigger: always_on
---

# Quy Chuẩn Lập Trình Chung (General Rules)

Tài liệu này định nghĩa các quy tắc chung nhất áp dụng cho tất cả các tác vụ lập trình và tương tác trong dự án. AI Agent phải tuân thủ nghiêm ngặt các hướng dẫn này.

---

## 📥 1. Phân Loại Yêu Cầu (Request Classifier)

Trước khi thực hiện **bất kỳ hành động nào**, AI Agent phải phân loại yêu cầu của người dùng theo bảng sau:

| Loại Yêu Cầu | Từ Khóa Nhận Diện | Cấp Độ Áp Dụng | Kết Quả Đầu Ra |
| :--- | :--- | :--- | :--- |
| **HỎI ĐÁP (Question)** | "là gì", "như thế nào", "giải thích" | Chỉ TIER 0 | Trả lời bằng văn bản (Text Response) |
| **KHẢO SÁT (Survey/Intel)** | "phân tích", "liệt kê file", "tổng quan" | TIER 0 + Explorer Agent | Session Intel (Không sửa file) |
| **CODE ĐƠN GIẢN (Simple)** | "sửa", "thêm", "thay đổi" (trên 1 file) | TIER 0 + TIER 1 (Lite) | Sửa trực tiếp (Inline Edit) |
| **CODE PHỨC TẠP (Complex)**| "xây dựng", "tạo mới", "triển khai", "refactor" | TIER 0 + TIER 1 (Full) + Agent chuyên sâu | **Bắt buộc tạo Plan & Task Checklist** |
| **THIẾT KẾ / UI** | "thiết kế", "giao diện", "trang", "dashboard" | TIER 0 + TIER 1 + Agent chuyên sâu | **Bắt buộc tạo Plan & Task Checklist** |

---

## 🤖 2. Giao Thức Định Tuyến Tác Nhân (Agent Routing Protocol)

**BẮT BUỘC:** Trước khi phản hồi bất kỳ yêu cầu code hoặc thiết kế nào, AI Agent phải tự động phân tích và chọn Agent chuyên môn phù hợp nhất:

1.  **Phân tích (Silent Analysis)**: Nhận diện domain công việc (Frontend, Backend, Security, PM, QA...) từ yêu cầu của người dùng.
2.  **Chọn Agent**: Chọn chuyên gia tương ứng trong thư mục `.agents/agent/`.
3.  **Thông báo cho Người dùng (MANDATORY)**: In ra câu thông báo đầu tiên trước khi trả lời:
    ```markdown
    🤖 **Applying knowledge of `@[tên-agent]`...**
    ```
4.  **Nạp Skill tương ứng**: Đọc thuộc tính `skills` trong frontmatter của agent đã chọn và nạp các chỉ dẫn liên quan từ `.agents/skills/`.

---

## TIER 0: QUY TẮC TOÀN CỤC (ALWAYS ACTIVE)

### 💬 3. Phong Cách Giao Tiếp (Communication Style)
*   **Ngắn gọn & Trọng tâm**: Không giải thích dông dài hoặc lặp lại code không cần thiết. Đưa ra giải pháp trực tiếp và giải thích lý do đằng sau các quyết định thiết kế phức tạp.
*   **Sử dụng Link Clickable**: Khi nhắc đến bất kỳ file, class, struct hoặc function nào trong mã nguồn, **bắt buộc** phải tạo liên kết bằng cú pháp markdown dạng: `[tên_file](file:///đường_dẫn_tuyệt_đối_đến_file)`. Ví dụ: [index.ts](file:///d:/work/ag-tool-kit/cli/src/index.ts).
*   **Ngôn ngữ**: Trả lời bằng tiếng Việt. Code comments và tên biến/hàm bắt buộc viết bằng tiếng Anh.

### 🧹 4. Quy Chuẩn Lập Trình (Clean Code)
*   **Nguyên lý Clean Code**: 
    *   Hàm chỉ làm một việc (Single Responsibility Principle).
    *   Đặt tên biến và hàm mang ý nghĩa mô tả rõ ràng, tránh viết tắt khó hiểu.
    *   Giữ tài liệu comment và docstring hiện có không liên quan đến thay đổi, không được xóa bừa bãi.
*   **Tránh Hardcode**: Tất cả cấu hình, khóa API, URL endpoint phải được đưa vào biến môi trường (`.env`) hoặc file cấu hình chung của hệ thống.
*   **Xử lý lỗi (Error Handling)**: 
    *   Luôn luôn kiểm tra lỗi trả về và log chi tiết. Không nuốt lỗi (silent fail).

### 📁 5. Sự Phụ Thuộc File (File Dependency Awareness)
*   Trước khi sửa bất kỳ file nào, bắt buộc phải kiểm tra sơ đồ phụ thuộc của file đó để tránh ảnh hưởng dây chuyền (regression).

### 🐙 6. Quy Quy Chuẩn Git & Commit
Hệ thống tuân thủ tiêu chuẩn **Conventional Commits**:
*   `feat`: Tính năng mới cho người dùng.
*   `fix`: Sửa lỗi (bug fix).
*   `docs`: Thay đổi tài liệu hướng dẫn.
*   `style`: Thay đổi format code (whitespace, semicolon...) không làm thay đổi logic code.
*   `refactor`: Thay đổi cấu trúc code không sửa bug hay thêm tính năng mới.
*   `perf`: Thay đổi cải thiện hiệu năng.
*   `test`: Viết thêm unit test hoặc sửa test có sẵn.
*   `chore`: Các tác vụ nhỏ khác (cập nhật dependency, cấu hình build...).

### 🧪 7. Quy Trình Giải Quyết Tác Vụ (Workflow Execution)
Mọi thay đổi phức tạp (phân loại COMPLEX CODE hoặc DESIGN/UI) đều phải tuân theo quy trình 3 bước:
1.  **Lập kế hoạch (Planning)**: Nghiên cứu mã nguồn hiện tại, đề xuất phương án và tạo `implementation_plan.md` để người dùng duyệt.
2.  **Triển khai (Execution)**: Viết code và liên tục cập nhật tiến độ qua file `task.md`.
3.  **Xác minh & Bàn giao (Verification & Walkthrough)**: Chạy kiểm thử tự động, kiểm tra thực tế và viết báo cáo bàn giao `walkthrough.md`.
