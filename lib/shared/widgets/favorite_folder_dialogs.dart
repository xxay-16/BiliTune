import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/theme/app_colors.dart';
import '../../core/theme/app_dimens.dart';
import '../../core/theme/app_typography.dart';
import '../../data/models/models.dart';
import 'app_toast.dart';
import '../../state/providers.dart';

Future<void> showCreateFavoriteFolderDialog(BuildContext context) {
  return showDialog<void>(
    context: context,
    builder: (_) => const _CreateFavoriteFolderDialog(),
  );
}

Future<bool?> showAddToFavoriteDialog(BuildContext context, Track track) {
  return showDialog<bool>(
    context: context,
    builder: (_) => _FavoriteFolderDialog(track: track),
  );
}

Future<bool?> showMoveToFavoriteDialog(
  BuildContext context,
  Track track, {
  required int fromMediaId,
}) {
  return showDialog<bool>(
    context: context,
    builder: (_) =>
        _FavoriteFolderDialog(track: track, fromMediaId: fromMediaId),
  );
}

class _CreateFavoriteFolderDialog extends ConsumerStatefulWidget {
  const _CreateFavoriteFolderDialog();

  @override
  ConsumerState<_CreateFavoriteFolderDialog> createState() =>
      _CreateFavoriteFolderDialogState();
}

class _CreateFavoriteFolderDialogState
    extends ConsumerState<_CreateFavoriteFolderDialog> {
  final _titleController = TextEditingController();
  final _introController = TextEditingController();
  bool _isPrivate = false;

  @override
  void dispose() {
    _titleController.dispose();
    _introController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final auth = ref.watch(authProvider);
    final library = ref.watch(libraryProvider);

    return Dialog(
      backgroundColor: colors.bgElevated,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 460),
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.s5),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '新建收藏夹',
                style: AppTypography.titleM.copyWith(color: colors.textPrimary),
              ),
              const SizedBox(height: AppSpacing.s4),
              TextField(
                controller: _titleController,
                autofocus: true,
                style: AppTypography.body.copyWith(color: colors.textPrimary),
                decoration: _inputDecoration(colors, '收藏夹标题'),
              ),
              const SizedBox(height: AppSpacing.s3),
              TextField(
                controller: _introController,
                minLines: 2,
                maxLines: 3,
                style: AppTypography.body.copyWith(color: colors.textPrimary),
                decoration: _inputDecoration(colors, '简介，可留空'),
              ),
              const SizedBox(height: AppSpacing.s3),
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                value: _isPrivate,
                activeColor: colors.brand,
                onChanged: (value) =>
                    setState(() => _isPrivate = value ?? false),
                title: Text(
                  '设为私密收藏夹',
                  style: AppTypography.body.copyWith(color: colors.textPrimary),
                ),
              ),
              if (library.errorMessage != null) ...[
                const SizedBox(height: AppSpacing.s2),
                Text(
                  library.errorMessage!,
                  style: AppTypography.caption.copyWith(color: colors.error),
                ),
              ],
              const SizedBox(height: AppSpacing.s4),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: library.isLoading
                          ? null
                          : () => Navigator.of(context).pop(),
                      child: const Text('取消'),
                    ),
                  ),
                  const SizedBox(width: AppSpacing.s3),
                  Expanded(
                    child: FilledButton(
                      onPressed: !auth.isSignedIn || library.isLoading
                          ? null
                          : _submit,
                      child: library.isLoading
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Text('创建'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  InputDecoration _inputDecoration(BiliColors colors, String hint) {
    return InputDecoration(
      hintText: hint,
      hintStyle: AppTypography.caption.copyWith(color: colors.textTertiary),
      filled: true,
      fillColor: colors.bgHighlight,
      border: OutlineInputBorder(
        borderRadius: AppRadius.smAll,
        borderSide: BorderSide.none,
      ),
    );
  }

  Future<void> _submit() async {
    await ref
        .read(libraryProvider.notifier)
        .createFavoriteFolder(
          title: _titleController.text,
          intro: _introController.text,
          isPrivate: _isPrivate,
        );
    if (!mounted) return;
    if (ref.read(libraryProvider).errorMessage == null) {
      Navigator.of(context).pop(true);
    }
  }
}

class _FavoriteFolderDialog extends ConsumerStatefulWidget {
  const _FavoriteFolderDialog({required this.track, this.fromMediaId});

  final Track track;
  final int? fromMediaId;

  @override
  ConsumerState<_FavoriteFolderDialog> createState() =>
      _FavoriteFolderDialogState();
}

class _FavoriteFolderDialogState extends ConsumerState<_FavoriteFolderDialog> {
  bool _isSubmitting = false;
  String? _errorMessage;

  bool get _isMoving => widget.fromMediaId != null;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final auth = ref.watch(authProvider);
    final library = ref.watch(libraryProvider);
    final folders = library.folders
        .where((folder) => folder.mediaId != widget.fromMediaId)
        .toList(growable: false);
    final errorMessage = _errorMessage ?? library.errorMessage;

    return Dialog(
      backgroundColor: colors.bgElevated,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480, maxHeight: 560),
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.s5),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      _isMoving ? '移动到其他收藏夹' : '添加到收藏夹',
                      style: AppTypography.titleM.copyWith(
                        color: colors.textPrimary,
                      ),
                    ),
                  ),
                  IconButton(
                    icon: Icon(
                      Icons.close_rounded,
                      color: colors.textSecondary,
                    ),
                    onPressed: _isSubmitting
                        ? null
                        : () => Navigator.of(context).pop(),
                  ),
                ],
              ),
              const SizedBox(height: AppSpacing.s2),
              Text(
                widget.track.title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: AppTypography.body.copyWith(color: colors.textSecondary),
              ),
              const SizedBox(height: AppSpacing.s4),
              if (_isSubmitting) ...[
                const LinearProgressIndicator(minHeight: 2),
                const SizedBox(height: AppSpacing.s3),
              ],
              if (!auth.isSignedIn)
                Text(
                  '需要先登录 Bilibili 账号。',
                  style: AppTypography.body.copyWith(
                    color: colors.textSecondary,
                  ),
                )
              else if (library.isLoading)
                const Center(child: CircularProgressIndicator())
              else if (folders.isEmpty)
                Text(
                  _isMoving ? '没有其他可用收藏夹，请先新建一个收藏夹。' : '还没有可用收藏夹，可以先新建一个。',
                  style: AppTypography.body.copyWith(
                    color: colors.textSecondary,
                  ),
                )
              else
                Flexible(
                  child: ListView.separated(
                    shrinkWrap: true,
                    itemCount: folders.length,
                    separatorBuilder: (_, _) =>
                        Divider(height: 1, color: colors.bgHighlight),
                    itemBuilder: (_, index) {
                      final folder = folders[index];
                      return ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: Icon(
                          folder.isPublic
                              ? Icons.folder_outlined
                              : Icons.lock_outline_rounded,
                          color: colors.textSecondary,
                        ),
                        title: Text(
                          folder.title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: AppTypography.body.copyWith(
                            color: colors.textPrimary,
                          ),
                        ),
                        subtitle: Text(
                          '${folder.mediaCount} 首',
                          style: AppTypography.caption.copyWith(
                            color: colors.textTertiary,
                          ),
                        ),
                        onTap: _isSubmitting || library.isUpdatingFavorites
                            ? null
                            : () => _submit(folder),
                      );
                    },
                  ),
                ),
              if (errorMessage != null) ...[
                const SizedBox(height: AppSpacing.s3),
                Text(
                  errorMessage,
                  style: AppTypography.caption.copyWith(color: colors.error),
                ),
              ],
              if (!_isMoving) ...[
                const SizedBox(height: AppSpacing.s4),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton.icon(
                    icon: const Icon(Icons.create_new_folder_outlined),
                    label: const Text('新建收藏夹'),
                    onPressed: auth.isSignedIn && !_isSubmitting
                        ? () => showCreateFavoriteFolderDialog(context)
                        : null,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _submit(BiliFavoriteFolder folder) async {
    if (_isSubmitting) return;
    setState(() {
      _isSubmitting = true;
      _errorMessage = null;
    });
    try {
      final notifier = ref.read(libraryProvider.notifier);
      if (_isMoving) {
        await notifier.moveTrackToFavoriteFolder(
          widget.track,
          fromMediaId: widget.fromMediaId!,
          toMediaId: folder.mediaId,
        );
      } else {
        await notifier.addTrackToFavoriteFolder(widget.track, folder.mediaId);
      }
      if (!mounted) return;
      final colors = context.colors;
      showAppToast(
        context,
        message: _isMoving ? '已移动到 ${folder.title}' : '已添加到 ${folder.title}',
        icon: _isMoving
            ? Icons.drive_file_move_outlined
            : Icons.favorite_rounded,
        accentColor: colors.brand,
      );
      Navigator.of(context).pop(true);
    } catch (error) {
      if (mounted) setState(() => _errorMessage = error.toString());
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }
}
