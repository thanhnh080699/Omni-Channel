---
name: nextjs-react-expert
description: Chuyên gia phát triển ứng dụng web React và Next.js. Cung cấp các best practices về Client/Server component, tối ưu hóa CSS, hydration và render.
when_to_use: "Dự án phát hiện có package.json chứa 'next' hoặc 'react' hoặc các file mở rộng .jsx, .tsx"
---

# Kỹ Năng: Next.js & React Expert

Chỉ dẫn chuyên sâu này được tự động nạp khi phát hiện dự án sử dụng Next.js hoặc React.

---

## 🎨 1. Thiết Kế Giao Diện & Thẩm Mỹ (UI/UX Aesthetics)

*   **Sử dụng Bảng màu phong phú (Rich Aesthetics)**: Tránh sử dụng các màu đơn điệu (đỏ, xanh thuần). Hãy sử dụng các bảng màu tinh tế, hiện đại (như hệ màu HSL, sleek dark mode, gradient mịn, hiệu ứng kính mờ - glassmorphism).
*   **Typography**: Sử dụng các font chữ hiện đại từ Google Fonts (như Inter, Roboto, Outfit) thay vì font chữ mặc định của trình duyệt.
*   **Hiệu ứng Động (Dynamic Designs)**: Tăng tương tác người dùng bằng hiệu ứng hover mượt mà và các micro-animation.
*   **Tuyệt đối không dùng Placeholder**: Khi cần hình ảnh minh họa, hãy sử dụng công cụ sinh ảnh (`generate_image`) thay vì chèn link trống hoặc ảnh placeholder xám.
*   **Responsive**: Mọi giao diện phải được tối ưu cho cả thiết bị di động (Mobile First), Tablet và Desktop.

---

## 🏗️ 2. Nguyên Tắc Thiết Kế Component (Server vs Client)

*   **Mặc định là Server Component**: Tất cả các component trong thư mục `app/` (Next.js App Router) mặc định là Server Component để tối ưu tải trang và SEO.
*   **Sử dụng Client Component đúng cách**:
    *   Chỉ thêm chỉ thị `"use client"` ở dòng đầu tiên của file khi component có sử dụng:
        *   Các hook tương tác (`useState`, `useEffect`, `useReducer`, `useRef`).
        *   Các API trình duyệt (`window`, `document`, `localStorage`).
        *   Các sự kiện người dùng (`onClick`, `onChange`, `onSubmit`).
*   **Tách biệt logic**: Đưa các phần tương tác client vào các component nhỏ nhất có thể, tránh khai báo `"use client"` ở các trang lớn (page.tsx) hoặc component layout gốc.
*   **State Management**: Ưu tiên sử dụng React Context cho các state nhỏ cục bộ và các thư viện như Zustand cho state toàn cục. Hạn chế tối đa việc prop-drilling quá sâu.

---

## ⚡ 3. Tối Ưu Hóa Render & Hydration

*   **Lỗi Hydration Mismatch**: Tránh sử dụng các API trình duyệt trực tiếp khi render HTML (ví dụ: `typeof window !== 'undefined'`, `new Date()`, `Math.random()`). Nếu bắt buộc phải dùng, hãy sử dụng `useEffect` để cập nhật trạng thái sau khi đã mount:
    ```tsx
    const [isMounted, setIsMounted] = useState(false);
    useEffect(() => { setIsMounted(true); }, []);
    if (!isMounted) return <Placeholder />;
    ```
*   **Lazy Loading (Tải chậm)**: Sử dụng `next/dynamic` cho các component client nặng hoặc các modal chỉ hiện lên khi click:
    ```tsx
    import dynamic from 'next/dynamic';
    const HeavyChart = dynamic(() => import('@/components/HeavyChart'), { ssr: false });
    ```
*   **Tối ưu hình ảnh**: Sử dụng thẻ `<Image>` của Next.js thay cho thẻ `<img>` thường để tự động nén và lười tải (lazy loading).

---

## 🔍 4. Tối Ưu Hóa SEO (SEO Best Practices)

Mọi trang web công cộng phải tự động tích hợp các tiêu chuẩn SEO sau:
*   **Title Tags**: Đặt thẻ tiêu đề độc nhất, mô tả ngắn gọn và chứa từ khóa chính cho từng trang.
*   **Meta Descriptions**: Thêm mô tả hấp dẫn dưới 160 ký tự để nâng cao tỷ lệ click (CTR).
*   **Cấu trúc Thẻ Headings**: Sử dụng đúng **duy nhất một** thẻ `<h1>` trên một trang và phân cấp hợp lý (`<h2>`, `<h3>`...).
*   **HTML Semantic**: Ưu tiên sử dụng các thẻ HTML5 có ngữ nghĩa như `<header>`, `<nav>`, `<main>`, `<article>`, `<footer>`, `<section>` thay vì lạm dụng thẻ `<div>`.
*   **Unique IDs**: Đảm bảo tất cả phần tử tương tác (nút bấm, form nhập liệu) có thuộc tính `id` duy nhất và rõ nghĩa để hỗ trợ test giao diện tự động.

---

## 🎨 5. Styling Best Practices

*   **CSS Modules**: Đặt tên file dạng `[ComponentName].module.css`. Tận dụng tối đa CSS Modules để tránh xung đột class và tăng tính tái sử dụng. Tránh import style trực tiếp không có module để tránh ô nhiễm style toàn cục.
*   **Tailwind CSS (Nếu sử dụng)**:
    *   Chỉ sử dụng khi dự án yêu cầu rõ ràng và đã thống nhất phiên bản sử dụng.
    *   Nhóm các class gọn gàng, tránh viết quá dài trên một dòng.
    *   Sử dụng thư viện `tailwind-merge` và `clsx` để gộp class động một cách an toàn.
