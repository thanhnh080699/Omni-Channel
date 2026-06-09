---
name: tailwind-css-expert
description: Chuyên gia thiết kế giao diện bằng Tailwind CSS. Hướng dẫn sử dụng utility classes, responsive breakpoints, dark mode, custom configuration, và kết hợp tailwind-merge/clsx.
when_to_use: "Dự án phát hiện có package.json chứa 'tailwindcss' hoặc có các file cấu hình tailwind.config.js, tailwind.config.ts hoặc imports tailwindcss trong CSS"
---

# Kỹ Năng: Tailwind CSS Expert

Chỉ dẫn chuyên sâu này được tự động nạp khi phát hiện dự án sử dụng Tailwind CSS để thiết kế giao diện.

---

## 🎨 1. Utility-First Mindset & Clean Classes

*   **Tránh Lạm Dụng `@apply`**: Chỉ sử dụng `@apply` trong file CSS cho các class tiện ích lặp lại cực kỳ nhiều (như custom scrollbar, typography của bên thứ ba). Ưu tiên viết utility classes trực tiếp trong HTML/React component để dễ bảo trì.
*   **Sắp Xếp Class Hợp Lý**: Sắp xếp class theo trình tự logic (Layout -> Box Model -> Typography -> Visual -> Interactive/State -> Responsive):
    *   *Ví dụ*: `flex items-center justify-between w-full p-4 text-white bg-blue-500 hover:bg-blue-600 md:p-6`
*   **Tránh Trùng Lặp & Ghi Đè Class**: Luôn sử dụng `tailwind-merge` kết hợp với `clsx` khi ghép class động để đảm bảo các class sau ghi đè chính xác class trước (không bị CSS cascade mặc định làm lỗi hiển thị):
    ```typescript
    import { twMerge } from 'tailwind-merge';
    import { clsx, ClassValue } from 'clsx';
    
    export function cn(...inputs: ClassValue[]) {
      return twMerge(clsx(inputs));
    }
    ```

---

## 📱 2. Responsive & State Prefixes

*   **Mobile First**: Thiết kế mặc định cho màn hình di động (không có prefix), sau đó thêm các breakpoint (`sm:`, `md:`, `lg:`, `xl:`) để tinh chỉnh giao diện trên màn hình lớn.
*   **Interactive States**: Tận dụng tối đa các state prefix như `hover:`, `focus:`, `active:`, `disabled:` để tạo micro-interaction.
*   **Dark Mode**: Sử dụng prefix `dark:` để định nghĩa màu sắc cho chế độ tối. Đảm bảo cấu hình dark mode trong tailwind config khớp với cơ chế chuyển đổi class của ứng dụng (ví dụ: selector `class` trên thẻ `<html>` hoặc `<body>`).

---

## ⚠️ 3. Quy Tắc Tránh Lỗi Biên Dịch (Purge/JIT Rules)

*   **Không Nối Chuỗi Class Động**: Tailwind CSS quét mã nguồn tĩnh để sinh CSS tương ứng. **Không bao giờ** viết class bằng cách ghép nối chuỗi động:
    *   *Sai*: `text-${color}-500` (Tailwind sẽ không nhận diện được để build).
    *   *Đúng*: Sử dụng object map hoặc viết đầy đủ tên class:
        ```typescript
        const colorClasses = {
          red: 'text-red-500',
          blue: 'text-blue-500',
        };
        ```
