import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:share_plus/share_plus.dart';

import '../../core/network/api_exception.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/share_api.dart';
import '../auth/wechat_auth_service.dart';

final _bundleProvider =
    FutureProvider.family<ShareBundle, String>((ref, id) async {
  return ref.read(shareApiProvider).bundle(id);
});

/// 分享海报：渲染主页推广卡片 + 落地页二维码，支持微信/系统分享。
class PosterScreen extends ConsumerStatefulWidget {
  const PosterScreen({super.key, required this.profileId});

  final String profileId;

  @override
  ConsumerState<PosterScreen> createState() => _PosterScreenState();
}

class _PosterScreenState extends ConsumerState<PosterScreen> {
  final _boundaryKey = GlobalKey();
  bool _busy = false;

  Future<Uint8List?> _capture() async {
    final boundary = _boundaryKey.currentContext?.findRenderObject()
        as RenderRepaintBoundary?;
    if (boundary == null) return null;
    final image = await boundary.toImage(pixelRatio: 3);
    final data = await image.toByteData(format: ui.ImageByteFormat.png);
    return data?.buffer.asUint8List();
  }

  Future<void> _shareWeChat({required bool timeline}) async {
    setState(() => _busy = true);
    try {
      final bytes = await _capture();
      if (bytes == null) return;
      await WeChatAuthService.instance.shareImage(bytes, timeline: timeline);
    } on ApiException catch (e) {
      _fallbackOrToast(e.message);
    } catch (e) {
      _toast('分享失败：$e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _shareSystem() async {
    setState(() => _busy = true);
    try {
      final bytes = await _capture();
      if (bytes == null) return;
      await Share.shareXFiles([
        XFile.fromData(bytes, name: 'weifou_poster.png', mimeType: 'image/png'),
      ]);
    } catch (e) {
      _toast('分享失败：$e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _fallbackOrToast(String msg) {
    // 微信未配置/未安装 → 提示并引导系统分享。
    _toast('$msg，可改用系统分享');
  }

  void _toast(String m) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(_bundleProvider(widget.profileId));
    return Scaffold(
      appBar: AppBar(title: const Text('分享海报')),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text('加载失败：$e')),
        data: (b) => Column(
          children: [
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Center(
                  child: RepaintBoundary(
                    key: _boundaryKey,
                    child: _PosterCard(bundle: b),
                  ),
                ),
              ),
            ),
            SafeArea(
              top: false,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    Expanded(
                      child: OutlinedButton(
                        onPressed: _busy
                            ? null
                            : () => _shareWeChat(timeline: false),
                        child: const Text('微信好友'),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: OutlinedButton(
                        onPressed:
                            _busy ? null : () => _shareWeChat(timeline: true),
                        child: const Text('朋友圈'),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: ElevatedButton(
                        onPressed: _busy ? null : _shareSystem,
                        child: const Text('更多'),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PosterCard extends StatelessWidget {
  const _PosterCard({required this.bundle});
  final ShareBundle bundle;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 300,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            bundle.realName ?? bundle.nickname ?? '',
            style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 12),
          if (bundle.oneLiner != null)
            Text(bundle.oneLiner!,
                style: const TextStyle(fontSize: 15, height: 1.5)),
          if (bundle.tags.isNotEmpty) ...[
            const SizedBox(height: 16),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                for (final t in bundle.tags)
                  Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      border: Border.all(color: AppTheme.border),
                      borderRadius: BorderRadius.circular(100),
                    ),
                    child: Text(t, style: const TextStyle(fontSize: 12)),
                  ),
              ],
            ),
          ],
          const SizedBox(height: 24),
          Center(
            child: QrImageView(
              data: bundle.shareUrl,
              version: QrVersions.auto,
              size: 140,
            ),
          ),
          const SizedBox(height: 8),
          const Center(
            child: Text('扫码：加微信前，先和我的 AI 聊聊',
                style: TextStyle(fontSize: 12, color: AppTheme.sub)),
          ),
        ],
      ),
    );
  }
}
