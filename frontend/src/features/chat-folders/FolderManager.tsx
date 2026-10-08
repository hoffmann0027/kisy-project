// Folder management modal (UPD3 stage H): create, rename, delete and
// reorder the user's chat folders. Chats are added to folders from the
// chat-list context menu.
import { useState } from "react";
import { t } from "@shared/i18n";
import { Button, Input, Modal, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import {
  useCreateFolder,
  useDeleteFolder,
  useFolders,
  useRenameFolder,
  useReorderFolders,
} from "@entities/chat-folders/queries";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function FolderManager({ open, onClose }: Props) {
  const { folders } = useFolders();
  const createFolder = useCreateFolder();
  const renameFolder = useRenameFolder();
  const deleteFolder = useDeleteFolder();
  const reorderFolders = useReorderFolders();

  const [newName, setNewName] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");

  const create = () => {
    const name = newName.trim();
    if (!name) return;
    createFolder.mutate(name, {
      onSuccess: () => setNewName(""),
      onError: () => toast.error(t("chat.folders.createFailed")),
    });
  };

  const saveRename = (id: string) => {
    const name = editName.trim();
    if (!name) return;
    renameFolder.mutate(
      { id, name },
      {
        onSuccess: () => setEditingId(null),
        onError: () => toast.error(t("chat.folders.renameFailed")),
      },
    );
  };

  const move = (index: number, dir: -1 | 1) => {
    const ids = folders.map((f) => f.id);
    const j = index + dir;
    if (j < 0 || j >= ids.length) return;
    [ids[index], ids[j]] = [ids[j], ids[index]];
    reorderFolders.mutate(ids, { onError: () => toast.error(t("chat.folders.reorderFailed")) });
  };

  return (
    <Modal open={open} title={t("chat.list.folders")} onClose={onClose}>
      <div className="foldermgr">
        <div className="foldermgr__create">
          <Input
            placeholder={t("chat.folders.newFolderPlaceholder")}
            value={newName}
            maxLength={64}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && create()}
          />
          <Button onClick={create} disabled={!newName.trim() || createFolder.isPending}>
            {t("chat.folders.create")}
          </Button>
        </div>

        {folders.length === 0 && (
          <p className="foldermgr__empty">{t("chat.folders.empty")}</p>
        )}

        <ul className="foldermgr__list">
          {folders.map((f, i) => (
            <li key={f.id} className="foldermgr__row">
              {editingId === f.id ? (
                <>
                  <Input
                    value={editName}
                    maxLength={64}
                    autoFocus
                    onChange={(e) => setEditName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") saveRename(f.id);
                      if (e.key === "Escape") setEditingId(null);
                    }}
                  />
                  <button className="foldermgr__btn" title={t("chat.action.save")} onClick={() => saveRename(f.id)}>
                    <Icon.Check size={16} />
                  </button>
                </>
              ) : (
                <>
                  <span className="foldermgr__name">
                    <Icon.Folder size={16} />
                    {f.name}
                    <span className="foldermgr__count">{f.items.length}</span>
                  </span>
                  <span className="foldermgr__actions">
                    <button className="foldermgr__btn" title={t("chat.folders.moveUp")} disabled={i === 0} onClick={() => move(i, -1)}>
                      ↑
                    </button>
                    <button
                      className="foldermgr__btn"
                      title={t("chat.folders.moveDown")}
                      disabled={i === folders.length - 1}
                      onClick={() => move(i, 1)}
                    >
                      ↓
                    </button>
                    <button
                      className="foldermgr__btn"
                      title={t("chat.folders.rename")}
                      onClick={() => {
                        setEditingId(f.id);
                        setEditName(f.name);
                      }}
                    >
                      <Icon.Edit size={16} />
                    </button>
                    <button
                      className="foldermgr__btn foldermgr__btn--danger"
                      title={t("chat.folders.delete")}
                      onClick={() =>
                        deleteFolder.mutate(f.id, { onError: () => toast.error(t("chat.folders.deleteFailed")) })
                      }
                    >
                      <Icon.Trash size={16} />
                    </button>
                  </span>
                </>
              )}
            </li>
          ))}
        </ul>
        <p className="foldermgr__hint">{t("chat.folders.hint")}</p>
      </div>
    </Modal>
  );
}
