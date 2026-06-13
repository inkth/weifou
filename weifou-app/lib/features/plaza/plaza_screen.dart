import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/app_theme.dart';
import '../../data/api/plaza_api.dart';

typedef _Query = ({String sort, String q});

final _plazaProvider =
    FutureProvider.family<List<PlazaCard>, _Query>((ref, query) async {
  return ref.read(plazaApiProvider).list(sort: query.sort, q: query.q);
});

/// 人物广场（发现 Tab）：浏览公开的真人 AI 分身。
class PlazaScreen extends ConsumerStatefulWidget {
  const PlazaScreen({super.key});

  @override
  ConsumerState<PlazaScreen> createState() => _PlazaScreenState();
}

class _PlazaScreenState extends ConsumerState<PlazaScreen> {
  String _sort = 'new';
  String _q = '';
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final query = (sort: _sort, q: _q);
    final async = ref.watch(_plazaProvider(query));
    return Scaffold(
      appBar: AppBar(
        title: const Text('广场'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(96),
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: TextField(
                  controller: _searchCtrl,
                  textInputAction: TextInputAction.search,
                  onSubmitted: (v) => setState(() => _q = v.trim()),
                  decoration: InputDecoration(
                    hintText: '搜索名字 / 方向 / 简介',
                    prefixIcon: const Icon(Icons.search),
                    isDense: true,
                    filled: true,
                    fillColor: Colors.white,
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(24),
                      borderSide: const BorderSide(color: AppTheme.border),
                    ),
                  ),
                ),
              ),
              Row(
                children: [
                  const SizedBox(width: 12),
                  _SortChip(
                    label: '最新',
                    selected: _sort == 'new',
                    onTap: () => setState(() => _sort = 'new'),
                  ),
                  const SizedBox(width: 8),
                  _SortChip(
                    label: '最热',
                    selected: _sort == 'hot',
                    onTap: () => setState(() => _sort = 'hot'),
                  ),
                ],
              ),
              const SizedBox(height: 6),
            ],
          ),
        ),
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(_plazaProvider(query)),
        child: async.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => ListView(
            children: [const SizedBox(height: 120), Center(child: Text('加载失败：$e'))],
          ),
          data: (cards) => cards.isEmpty
              ? ListView(
                  children: const [
                    SizedBox(height: 120),
                    Center(
                      child: Text('还没有公开的 AI 分身',
                          style: TextStyle(color: AppTheme.sub)),
                    ),
                  ],
                )
              : ListView.separated(
                  padding: const EdgeInsets.all(16),
                  itemCount: cards.length,
                  separatorBuilder: (_, _) => const SizedBox(height: 12),
                  itemBuilder: (_, i) => _Card(card: cards[i]),
                ),
        ),
      ),
    );
  }
}

class _SortChip extends StatelessWidget {
  const _SortChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });
  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ChoiceChip(
      label: Text(label),
      selected: selected,
      onSelected: (_) => onTap(),
    );
  }
}

class _Card extends StatelessWidget {
  const _Card({required this.card});
  final PlazaCard card;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(16),
      onTap: () => context.pushNamed('profile',
          pathParameters: {'id': card.profileId}),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: AppTheme.border),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            CircleAvatar(
              radius: 26,
              backgroundColor: AppTheme.bg,
              backgroundImage: (card.avatarUrl != null &&
                      card.avatarUrl!.isNotEmpty)
                  ? NetworkImage(card.avatarUrl!)
                  : null,
              child: (card.avatarUrl == null || card.avatarUrl!.isEmpty)
                  ? Text(
                      card.realName.isNotEmpty ? card.realName[0] : '?',
                      style: const TextStyle(
                          fontSize: 20, fontWeight: FontWeight.w600),
                    )
                  : null,
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Text(card.realName,
                          style: const TextStyle(
                              fontSize: 16, fontWeight: FontWeight.w700)),
                      if (card.title != null && card.title!.isNotEmpty) ...[
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(card.title!,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                  color: AppTheme.sub, fontSize: 13)),
                        ),
                      ],
                    ],
                  ),
                  if (card.oneLiner != null && card.oneLiner!.isNotEmpty) ...[
                    const SizedBox(height: 6),
                    Text(card.oneLiner!,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(height: 1.4)),
                  ],
                  if (card.tags.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 6,
                      runSpacing: 6,
                      children: [
                        for (final t in card.tags.take(3))
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              color: AppTheme.bg,
                              borderRadius: BorderRadius.circular(100),
                            ),
                            child: Text(t,
                                style: const TextStyle(
                                    fontSize: 11, color: AppTheme.sub)),
                          ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
