import 'dart:io' show Platform;

/// 付费入口类型。
enum PayEntry {
  /// 虚拟权益类（解锁数字内容/额度等）：iOS 必须走 IAP，故隐藏。预留。
  virtualGoods,
}

/// iOS 合规网关——前端主控制点（审核看的是 UI 上有没有出现违规入口）。
///
/// 这是「入口能力表」的本地默认值；后续可由后端 `GET /api/config/entries`
/// 下发覆盖，使合规策略变更不必发版过审（见计划第五节）。后端 payment
/// handler 另有 X-Platform 兜底。
class EntryGate {
  EntryGate._();

  /// 各入口在 iOS / Android 的默认可见性。
  static const Map<PayEntry, ({bool ios, bool android})> _defaults = {
    PayEntry.virtualGoods: (ios: false, android: true),
  };

  /// 后端下发的覆盖值（key 为 PayEntry.name）。为空时用本地默认。
  static Map<String, bool>? _remoteOverride;

  /// 应用后端 entries 接口的结果。
  static void applyRemote(Map<String, bool> override) {
    _remoteOverride = override;
  }

  /// 当前平台下该入口是否可见。
  static bool isVisible(PayEntry entry) {
    final remote = _remoteOverride?[entry.name];
    if (remote != null) return remote;
    final d = _defaults[entry]!;
    return Platform.isIOS ? d.ios : d.android;
  }
}
