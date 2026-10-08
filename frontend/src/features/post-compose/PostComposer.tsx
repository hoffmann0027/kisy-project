import { useRef, useState } from "react";
import { Button, IconButton, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { userFacingError } from "@shared/api/envelope";
import { useCreatePost } from "@entities/post/queries";
import { useAuthStore } from "@shared/store/auth";
import { fileTooLargeForNewAccount, heldBackNotice } from "@shared/lib/quarantine";
import { t } from "@shared/i18n";

// Writing a post: text plus up to ten files.
//
// The button stays disabled until the upload finishes rather than optimistically
// clearing: media travels after the post, so letting the author walk away early
// is letting them publish half of what they wrote.

const MAX_FILES = 10;

export function PostComposer({ communityId }: { communityId: string }) {
  const [text, setText] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const fileInput = useRef<HTMLInputElement>(null);
  const create = useCreatePost();
  // A brand-new account may not publish yet. Saying so here beats letting
  // someone write a post and lose it to a refusal on submit.
  const quarantine = useAuthStore((s) => s.quarantine);
  const held = heldBackNotice(quarantine, t("work.compose.heldBack"));

  const pick = (list: FileList | null) => {
    if (!list) return;
    const tooBig = Array.from(list)
      .map((f) => fileTooLargeForNewAccount(quarantine, f))
      .find(Boolean);
    if (tooBig) {
      toast.error(tooBig);
      return;
    }
    const picked = [...files, ...Array.from(list)].slice(0, MAX_FILES);
    if (picked.length < files.length + list.length) {
      toast.error(t("work.compose.tooManyFiles", { max: MAX_FILES }));
    }
    setFiles(picked);
  };

  const publish = () => {
    if (!text.trim() && files.length === 0) return;
    create.mutate(
      { communityId, text, files },
      {
        onSuccess: () => {
          setText("");
          setFiles([]);
          if (fileInput.current) fileInput.current.value = "";
        },
        onError: (e) => toast.error(userFacingError(e, t("work.compose.publishFailed"))),
      },
    );
  };

  if (held) {
    return <p className="post-compose__held">{held}</p>;
  }

  return (
    <div className="post-compose">
      <textarea
        className="post-compose__text"
        placeholder={t("work.compose.placeholder")}
        value={text}
        maxLength={8000}
        onChange={(e) => setText(e.target.value)}
      />
      <div className="post-compose__row">
        <IconButton label={t("work.compose.attach")} onClick={() => fileInput.current?.click()}>
          <Icon.Paperclip size={20} />
        </IconButton>
        <input
          ref={fileInput}
          type="file"
          multiple
          hidden
          onChange={(e) => pick(e.target.files)}
        />
        <span className="post-compose__files">
          {files.length > 0 ? files.map((f) => f.name).join(", ") : ""}
        </span>
        <Button
          onClick={publish}
          loading={create.isPending}
          disabled={!text.trim() && files.length === 0}
        >
          {t("work.compose.publish")}
        </Button>
      </div>
    </div>
  );
}
