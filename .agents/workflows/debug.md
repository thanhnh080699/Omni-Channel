# Workflow Command: /debug

Quy trình gỡ lỗi và phân tích nguyên nhân gốc rễ (Root Cause Analysis - RCA) một cách hệ thống.

---

## 📋 Mô Tả
Lệnh `/debug` được kích hoạt khi phát hiện lỗi hệ thống, kiểm thử thất bại (failed tests) hoặc ứng dụng hoạt động không đúng kỳ vọng. Quy trình này giúp định vị lỗi nhanh chóng và đưa ra giải pháp khắc phục an toàn nhất.

---

## 🔄 Các Bước Thực Hiện (Execution Steps)

### Bước 1: Thu Thập Thông Tin & Tái Hiện (Information Gathering)
1.  Đọc kỹ log lỗi (stack trace), mã lỗi (status codes) và ghi nhận thời điểm xảy ra lỗi.
2.  Xác định các điều kiện đầu vào (input), trạng thái hệ thống (state) dẫn đến việc lỗi xuất hiện.
3.  Thử tái hiện lỗi bằng cách chạy script test hoặc gọi thử API/CLI tương ứng.

### Bước 2: Phân Tích Nguyên Nhân (Root Cause Analysis)
1.  Truy vết mã nguồn dọc theo luồng dữ liệu (data flow) từ điểm báo lỗi ngược về nguồn phát sinh.
2.  Đưa ra các giả thuyết (hypotheses) về nguyên nhân gây lỗi.
3.  Kiểm tra các giả thuyết bằng cách đọc code hoặc chèn log bổ sung (debug logging) để kiểm tra giá trị biến.

### Bước 3: Đề Xuất & Sửa Lỗi (Remediation)
1.  Trình bày nguyên nhân lỗi và đề xuất các phương án khắc phục (phân tích ưu/nhược điểm từng cách).
2.  Sau khi người dùng đồng ý phương án, thực hiện sửa đổi mã nguồn.
3.  Chạy lại các test liên quan để đảm bảo lỗi đã được khắc phục hoàn toàn và không phát sinh lỗi mới (regression).
