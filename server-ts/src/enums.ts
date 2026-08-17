// 枚举契约: 与 docs/contract/terms.md、fields.md 严格对齐。
export type RegionLevel = 1 | 2 | 3 | 4; // 1集团 2大区 3省 4城市
export type AddressLevel = 1 | 2 | 3 | 4 | 5; // 1市 2区 3街道 4小区 5楼栋

export type RoleCode =
  | 'customer' | 'technician' | 'asset_admin'
  | 'resource_admin' | 'ops' | 'analyst' | 'sysadmin';

export type AccountStatus = 1 | 0; // 1启用 0停用

export type ProductStatus = 'DRAFT' | 'PUBLISHED' | 'OFFLINE';

export type RealNameStatus = 'VERIFIED' | 'PENDING';
export type ServiceStatus = 'ACTIVE' | 'ARREARS' | 'SUSPENDED';

export type OrderStatus = 'PENDING' | 'RESERVED' | 'INSTALLING' | 'DONE' | 'CANCELLED';

export type BillStatus = 'UNPAID' | 'PAID' | 'OVERDUE';
export type PaymentStatus = 'SUCCESS' | 'FAILED' | 'REFUNDED';

export type AssetStatus = 'IN_STOCK' | 'DEPLOYED' | 'MAINTENANCE' | 'SCRAPPED';

export type PortStatus = 'IDLE' | 'RESERVED' | 'USED' | 'DISABLED';

export type QuadStatus = 'LINKED' | 'CONFLICT' | 'UNLINKED';

export type TagStatus = 'UNBOUND' | 'BOUND' | 'DISABLED';

export type ResourceStatus = 'ONLINE' | 'OFFLINE' | 'FAULT';
export type LoAccountStatus = 'ACTIVE' | 'SUSPENDED' | 'CLOSED';

export type TicketStatus = 'PENDING' | 'DOING' | 'DONE' | 'CANCELED';
export type StageResult = 'PENDING' | 'DOING' | 'DONE';

export type TaskStatus = 'PENDING' | 'DOING' | 'DONE' | 'FAILED';

export type ComplaintStatus = 'OPEN' | 'PROCESSING' | 'CLOSED';
export type ScanResult = 'MATCH' | 'MISMATCH' | 'OFFLINE_CACHED';
export type ActivationResult = 'PENDING' | 'SUCCESS' | 'FAILED';
export type MessageLevel = 'INFO' | 'WARN' | 'URGENT';
export type PaymentMethod = 'wechat' | 'alipay' | 'card' | 'cash';
