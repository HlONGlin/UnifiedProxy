<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy kết hợp bảng điều khiển Mihomo gốc với quản lý đội nút trong các chế độ độc lập, điều khiển và Agent. Quản lý inbound, máy khách, đăng ký, tác vụ ký số, Sidecar, chứng chỉ, sao lưu WebDAV và cập nhật bằng 11 ngôn ngữ. Năm bảng màu độc lập với chế độ sáng/tối; mặc định là T0 đen trắng và sáng.

Bắt đầu với [triển khai và di chuyển](docs/unifiedproxy-migration.md). Bản sao lưu cũ giữ thông tin đăng nhập, lưu lượng và bí danh đăng ký; yêu cầu về tệp chứng chỉ, đăng ký lại nút và ngoại lệ NaïveProxy có trong [báo cáo tương thích](docs/design/legacy-acceptance.md).

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

<a id="deploy"></a>

## Cài đặt trên Linux

Chạy với quyền `root` trên Linux dùng systemd hoặc OpenRC:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh)
```

Trình cài đặt chọn một cổng trống có năm chữ số, tạo đường dẫn bảng điều khiển 16 ký tự, tên người dùng 14 ký tự và mật khẩu 16 ký tự. Địa chỉ truy cập và thông tin đăng nhập chỉ được in một lần khi kết thúc. Bạn cũng có thể tự chỉ định:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy được cài vào `/usr/local/unifiedproxy`, dữ liệu nằm trong `/usr/local/unifiedproxy/data`. Trình cài đặt cũng tải lõi Mihomo cho kiến trúc hiện tại; lõi đã có sẽ được giữ lại trừ khi bạn dùng `--update-mihomo`.

`unifiedproxy update` nâng cấp UnifiedProxy và giữ nguyên mọi cài đặt. Trước khi cập nhật, nó chụp nhanh thư mục dữ liệu vào `data/backups/` và giữ 3 bản mới nhất.

Gói UnifiedProxy dùng tên `unifiedproxy-linux-<architecture>.tar.gz` cho `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7`, `s390x` và `unifiedproxy-windows-amd64.zip`. dịch vụ và lệnh quản lý tiếp tục dùng tên tương thích `unifiedproxy`.

<details>
<summary>Docker</summary>

Image Docker dành cho Linux amd64, chứa UnifiedProxy và một lõi Mihomo có phiên bản cố định, đã được kiểm tra. Compose dùng mạng của máy chủ, nên cổng của các inbound thêm trong bảng điều khiển hoạt động ngay mà không cần ánh xạ cổng.

```bash
git clone https://github.com/HlONGlin/unifiedproxy.git
cd unifiedproxy
cp .env.example .env
docker compose up -d --build
docker compose logs unifiedproxy
```

Xây dựng từ kho này. Các bản phát hành gắn thẻ có thể xuất bản `ghcr.io/hlonglin/unifiedproxy:<tag>` và `latest`; chỉ dùng `docker compose pull` khi ảnh UnifiedProxy tương ứng đã có. `UNIFIEDPROXY_PASSWORD` / `UNIFIEDPROXY_PATH` để trống được tạo ở lần chạy đầu và in một lần vào log. Giữ tên dịch vụ `unifiedproxy` và các volume `unifiedproxy-data`, `unifiedproxy-core` khi nâng cấp để bảo toàn dữ liệu.

</details>

<a id="workspace"></a>

## Ảnh chụp màn hình

![Đội nút](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>Ảnh chụp màn hình</summary>

**Inbound**

![Inbound](./media/screenshots/unifiedproxy/inbounds.png)

**Hạn mức và hết hạn**

![Hạn mức và hết hạn](./media/screenshots/unifiedproxy/client-edit.png)

**Outbound**

![Outbound](./media/screenshots/unifiedproxy/outbounds.png)

**Định tuyến**

![Định tuyến](./media/screenshots/unifiedproxy/routing.png)

**Nhập Xray / 3x-ui**

![Nhập Xray / 3x-ui](./media/screenshots/unifiedproxy/xray-import.png)

**Đăng nhập**

![Đăng nhập](./media/screenshots/unifiedproxy/login.png)

**Đăng ký di động**

![Đăng ký di động](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

### Năm bảng màu

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | Đen trắng (mặc định) | Vercel |
| T1 | Giấy ấm và đất nung | Claude / Anthropic |
| T2 | Tím đêm | Sentry |
| T3 | Ngọc lục bảo | Supabase |
| T4 | Chàm | Stripe |

[Tokens / references](docs/design/p2-token-map.md) · [NOTICE](NOTICE)

<details>
<summary>T0–T4 · light / dark</summary>

| | light | dark |
|---|---|---|
| T0 | <img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 dark" width="240"> |
| T1 | <img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 dark" width="240"> |
| T2 | <img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 dark" width="240"> |
| T3 | <img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 dark" width="240"> |
| T4 | <img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 dark" width="240"> |

</details>

<a id="capabilities"></a>

## Tính năng

<details>
<summary>Multi-control</summary>

* Mạng P2P giữa các bảng điều khiển, không có nút chủ, dùng danh tính có chữ ký
* Đồng bộ inbound được mã hóa giữa các bảng điều khiển tin cậy
* Gộp đăng ký từ nhiều bảng điều khiển
* Thiết bị khách: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) nhập tệp mạng, đăng nhập một lần bằng tài khoản quản trị panel, sau đó xem được mọi panel và lấy subscription của từng panel hoặc subscription hợp nhất qua bất kỳ panel nào kết nối được

</details>

<details>
<summary>Đăng ký</summary>

* Trang đăng ký trên trình duyệt hiển thị mức dùng, hạn mức, thời hạn và các nút, kèm nút nhập vào Clash Caesar bằng một cú nhấp
* **Mẫu đăng ký:** mỗi đăng ký là một cấu hình client hoàn chỉnh, tạo từ mẫu tích hợp (Trung Quốc / Nga / Iran) hoặc mẫu Mihomo của riêng bạn
* Cấu hình đầy đủ cho Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X và Egern; danh sách nút cho V2Box, V2RayNG, Shadowrocket và các client dùng liên kết khác
* Token đăng ký riêng cho từng người dùng và cổng đăng ký riêng (tùy chọn)

</details>

<details>
<summary>Inbound và client</summary>

* Các listener `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` và `hysteria2-realm`
* Cài đặt TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya và che giấu lưu lượng
* Thông tin xác thực, hạn mức lưu lượng, thời hạn, lịch đặt lại, tự động gia hạn và trạng thái trực tuyến cho từng client; có cả giới hạn ở cấp inbound
* Liên kết chia sẻ, mã QR, nhập inbound từ YAML listener của Mihomo, đặt lại lưu lượng hàng loạt

</details>

<details>
<summary>Cài đặt Mihomo</summary>

* Basics: chế độ định tuyến, phiên bản IP cho kết nối trực tiếp, IPv6, TCP đồng thời, thống kê, nhật ký và định tuyến cơ bản
* Outbound (gồm Snell, WireGuard và OpenVPN) và nhóm chính sách, với biểu mẫu, YAML gốc và chuyển đổi liên kết
* Quy tắc định tuyến được xét từ trên xuống dưới, kể cả quy tắc `AND`
* Outbound Cloudflare WARP và lối ra IP dân dụng VPNGate theo quốc gia và nhà mạng

</details>

<details>
<summary>Bảng điều khiển và Mihomo</summary>

* Trang tổng quan hiển thị trạng thái hệ thống, lưu lượng, kết nối và Mihomo; Mihomo tự khởi động cùng bảng điều khiển
* Tạo và kiểm tra YAML gốc của Mihomo, xem trước và tải xuống cấu hình
* Chuyển phiên bản Mihomo từ bản phát hành chính thức của MetaCubeX, cập nhật GeoIP / GeoSite / MetaDB
* Sao lưu và khôi phục đầy đủ: cài đặt, inbound, danh tính Multi-control (tùy chọn), bộ đệm liên bảng, mẫu đăng ký và thiết bị client của Multi-control
* **Chuyển từ 3x-ui chỉ bằng một cú nhấp:** nhập `x-ui.db` (v2.9.3–v3.10.2) hoặc `config.json` của Xray; inbound, máy khách cùng hạn mức, ngày hết hạn và lưu lượng, outbound, bộ cân bằng tải và quy tắc định tuyến được thêm vào cạnh những gì bảng điều khiển đang có, còn mọi thứ Mihomo không làm được được liệt kê trong nhật ký chuyển đổi có thể tải về
* Giao diện tiếng Anh, tiếng Trung giản thể, tiếng Trung phồn thể, tiếng Nhật, tiếng Nga, tiếng Ba Tư, tiếng Việt, tiếng Tây Ban Nha, tiếng Thổ Nhĩ Kỳ, tiếng Ukraina và tiếng Bồ Đào Nha (Brazil)

</details>

<a id="operations"></a>

## Chứng chỉ TLS

Lần cài đặt tương tác đầu tiên cho phép chọn chứng chỉ Let's Encrypt cho tên miền, chứng chỉ Let's Encrypt ngắn hạn cho IPv4, chứng chỉ có sẵn, hoặc bỏ qua TLS. Sau khi cài đặt, chạy:

```bash
unifiedproxy ssl
```

Menu có thể cấp chứng chỉ cho tên miền và IP, dùng API DNS của Cloudflare để cấp chứng chỉ tên miền và wildcard khi cổng 80 không dùng được, gia hạn, thu hồi, liệt kê và áp dụng chứng chỉ có sẵn. Nếu tệp chứng chỉ chỉ có chứng chỉ lá, UnifiedProxy sẽ tự bổ sung chuỗi chứng chỉ, vì các client như Mihomo Party và FlClash từ chối loại chứng chỉ này.

## Lệnh quản lý

```text
UnifiedProxy                  # mở trình đơn tương tác
unifiedproxy start | stop | restart | status | logs
unifiedproxy settings | configure | set-port
unifiedproxy reset-credentials | reset-path
unifiedproxy update | update-menu | ssl
unifiedproxy enable | disable | clear-logs | uninstall
unifiedproxy --lang <code>    ngôn ngữ trình đơn, ví dụ: unifiedproxy --lang vi
```

Menu hỗ trợ 11 ngôn ngữ giống bảng điều khiển và ghi nhớ lựa chọn của bạn. Trình cài đặt vẫn dùng tiếng Anh.

<a id="docs"></a>

## Tài liệu

* [Triển khai và di chuyển](docs/unifiedproxy-migration.md)
* [Mô hình và giao thức](docs/fleet-protocol.md)
* [Chức năng và tương thích](docs/design/legacy-acceptance.md)
* [Chủ đề và nguồn tham khảo](docs/design/p2-token-map.md)
* [Đánh giá P5](docs/design/p5-acceptance.md)
* [Bản phát hành](https://github.com/HlONGlin/unifiedproxy/releases)

## Phát triển

Chỉ xây dựng, kiểm thử và xem trước trong container Linux orca-linux tại `/workspace`. Windows chỉ dùng để sửa tệp và Git.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`UNIFIEDPROXY_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Lời cảm ơn

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): lõi proxy và mô hình cấu hình
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): mật độ thông tin và cách thao tác của giao diện
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): hành vi tham chiếu cho các định dạng đăng ký
* UnifiedProxy (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Tham chiếu bảng màu: Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe. Tài liệu do VoltAgent/awesome-design-md tổng hợp theo giấy phép MIT; xem [ghi chú thiết kế giao diện](docs/design/p2-token-map.md).

## Giấy phép

UnifiedProxy được phát hành theo [GNU General Public License version 3](LICENSE). Mã chuyển đổi liên kết của Mihomo giữ ghi nhận GPL-3.0 trong [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md).

> [!IMPORTANT]
> Chỉ sử dụng dự án này trên máy chủ và mạng mà bạn sở hữu hoặc được phép quản trị. Hãy tuân thủ luật pháp sở tại, điều khoản của nhà cung cấp và chính sách của các dịch vụ bạn truy cập.

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
