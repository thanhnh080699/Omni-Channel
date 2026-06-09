# Kiến Trúc AG Tool Kit (.agents)

Tài liệu này mô tả chi tiết kiến trúc thư mục `.agents/` và vai trò của từng thành phần trong việc hướng dẫn, mở rộng khả năng và kiểm soát chất lượng của AI Agent.

---

## 🏗️ Cấu Trúc Thư Mục

Thư mục `.agents/` được tổ chức thành các module chuyên biệt:

```plaintext
.agents/
├── ARCHITECTURE.md          # Tài liệu mô tả kiến trúc (file này)
├── agent/                  # Định nghĩa Persona cho 20 Tác nhân chuyên gia
├── skills/                  # Các module tri thức kỹ năng (Conditional Loading)
├── workflows/               # Kịch bản các quy trình làm việc tương tác (/plan, /debug...)
├── rules/                   # Quy chuẩn lập trình chung và riêng
├── memory/                  # Bộ nhớ dự án (Persistent Memory)
└── scripts/                 # Kịch bản kiểm soát chất lượng tự động
```

---

## 🤖 1. Tác Nhân Chuyên Gia (Specialist Agents)
Hệ thống định nghĩa 20 Agent chuyên biệt, mỗi Agent đại diện cho một vai trò chuyên môn trong đội ngũ phát triển phần mềm:
*   **`orchestrator`**: Điều phối chính, phân loại yêu cầu và gọi các agent chuyên sâu.
*   **`project-planner`**: Lập kế hoạch dự án, phân rã công việc.
*   **`frontend-specialist`**: Chuyên gia thiết kế và lập trình giao diện Web.
*   **`backend-specialist`**: Thiết kế cơ sở dữ liệu, API, và logic xử lý nghiệp vụ.
*   **`security-auditor`**: Kiểm tra mã nguồn, phát hiện lỗ hổng và rủi ro bảo mật.
*   *Và các agent khác như `debugger`, `test-engineer`, `seo-specialist`, `devops-engineer`...*

---

## 🧩 2. Kỹ Năng Tải Có Điều Kiện (Conditional Skills)
Các kỹ năng (`skills/`) chứa kiến thức công nghệ cụ thể (ví dụ: React, Next.js, FastAPI, SQL). Mỗi kỹ năng chứa file `SKILL.md` đi kèm chỉ dẫn chuyên sâu và một thuộc tính `when_to_use` ở phần YAML frontmatter.
*   **Cơ chế hoạt động**: AI Agent chỉ nạp tài liệu hướng dẫn của Skill đó vào ngữ cảnh khi phát hiện dự án sử dụng công nghệ tương ứng (ví dụ: thấy file `package.json` có `react` -> tự động nạp skill `nextjs-react-expert`). Việc này giúp giữ cho ngữ cảnh gọn gàng, giảm thiểu chi phí token và tăng độ chính xác của AI.

---

## 🔄 3. Quy Trình Làm Việc (Interactive Workflows / Slash Commands)
Workflows là các quy trình thực hiện công việc được chuẩn hóa dưới dạng các bước tương tác. Khi bạn gõ các lệnh như `/plan` hoặc `/debug`, Agent sẽ tìm đọc file tương ứng trong thư mục `workflows/` và tuân thủ các bước:
*   **`/plan`**: Thu thập thông tin, nghiên cứu mã nguồn, đưa ra tài liệu thiết kế và yêu cầu phê duyệt.
*   **`/debug`**: Phân tích lỗi một cách có hệ thống, đề xuất các giả thuyết và chạy thử nghiệm.
*   **`/verify`**: Thực hiện chạy thử nghiệm kiểm soát chất lượng từ linting đến test coverage.

---

## 🧠 4. Bộ Nhớ Dài Hạn (Persistent Memory)
Nằm trong thư mục `memory/`, file `MEMORY.md` đóng vai trò là "nhật ký kiến trúc" của dự án. File này được cấu trúc theo phân loại 4 nhóm (4-type taxonomy):
1.  **Project Context**: Tổng quan dự án, công nghệ cốt lõi và mục tiêu.
2.  **Architectural Decisions**: Các quyết định thiết kế kiến trúc đã chốt để tránh AI đề xuất thay đổi ngược lại.
3.  **Learnings & Gotchas**: Tổng hợp các lỗi đặc thù của dự án và cách tránh.
4.  **Actionable Rules**: Các quy tắc tùy biến bổ sung cho AI trong dự án hiện tại.

---

## 🛠️ 5. Script Xác Minh Chất Lượng (Validation Layer)
Nằm trong thư mục `scripts/`, các script tự động hóa (như Python hoặc Node.js) giúp tự động hóa quá trình kiểm thử mã nguồn trước khi bàn giao:
*   **`checklist.py`**: Quét nhanh mã nguồn để phát hiện lỗ hổng bảo mật sơ bộ (quên API key, hardcode mật khẩu), kiểm tra format code.
*   **`verify_all.py`**: Quy trình kiểm tra toàn diện trước khi release (chạy unit test, kiểm tra tính tương thích, đo hiệu năng).
