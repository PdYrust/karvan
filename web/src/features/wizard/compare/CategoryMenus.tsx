import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { Input } from '@cloudflare/kumo/components/input';
import { DotsThreeIcon, PencilSimpleIcon, PlusIcon, TrashIcon } from '@phosphor-icons/react';
import { useState, type FormEvent } from 'react';
import { useT } from '../../../i18n/locale';
import {
  addCategory,
  newCategoryId,
  removeCategory,
  renameCategory,
  type PlanNode,
} from '../../../lib/plan/plan';
import { presetConditions, presetKeys, type PresetKey } from '../../../lib/plan/presets';
import { useWizard } from '../wizard-context';
import { compareMessages, type CompareKey } from './messages';

const presetWord = (k: PresetKey) => `preset_${k}` as CompareKey;

/** Adds a ready-made category in one click, at the top or inside the open one. */
export function AddCategory({ parent }: { parent: PlanNode | undefined }) {
  const t = useT(compareMessages);
  const { plan } = useWizard();
  const add = (k: PresetKey) => {
    const id = newCategoryId(plan.plan?.tree ?? null, k);
    plan.edit(
      addCategory(id, {
        name: t(presetWord(k)),
        conditions: [presetConditions[k]],
        parentId: parent?.id,
      }),
    );
  };
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button icon={<PlusIcon size={16} />} className="pointer-coarse:h-11">
            <bdi>{parent ? t('addInside', { name: parent.name }) : t('addCategory')}</bdi>
          </Button>
        }
      />
      <DropdownMenu.Content align="start" className="min-w-60">
        {presetKeys.map((k) => (
          <DropdownMenu.Item key={k} onClick={() => add(k)}>
            {t(presetWord(k))}
          </DropdownMenu.Item>
        ))}
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

function RenameDialog({
  node,
  open,
  onOpenChange,
}: {
  node: PlanNode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT(compareMessages);
  const { plan } = useWizard();
  const [name, setName] = useState(node.name);
  const id = `rename-${node.id}`;
  const close = (next: boolean) => {
    if (!next) setName(node.name);
    onOpenChange(next);
  };
  const save = (e: FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    plan.edit(renameCategory(node.id, name.trim()));
    onOpenChange(false);
  };
  return (
    <Dialog.Root open={open} onOpenChange={close}>
      <Dialog className="p-6" size="base">
        <form onSubmit={save} className="flex flex-col gap-4">
          <Dialog.Title className="text-heading font-semibold">{t('renameTitle')}</Dialog.Title>
          <div className="flex flex-col gap-2">
            <label id={`${id}-label`} htmlFor={id} className="text-sm font-medium text-kumo-strong">
              {t('nameLabel')}
            </label>
            <Input
              id={id}
              aria-labelledby={`${id}-label`}
              autoComplete="off"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="pointer-coarse:h-11"
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button className="pointer-coarse:h-11" onClick={() => close(false)}>
              {t('cancel')}
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={!name.trim()}
              className="pointer-coarse:h-11"
            >
              {t('save')}
            </Button>
          </div>
        </form>
      </Dialog>
    </Dialog.Root>
  );
}

/** The "..." menu of one category: rename it, or remove it with what is inside. */
export function CategoryMenu({ node, onRemoved }: { node: PlanNode; onRemoved: () => void }) {
  const t = useT(compareMessages);
  const { plan } = useWizard();
  const [renaming, setRenaming] = useState(false);
  const label = t('categoryMenu', { name: node.name });
  return (
    <>
      <DropdownMenu>
        <DropdownMenu.Trigger
          render={
            <Button
              variant="ghost"
              shape="square"
              size="sm"
              aria-label={label}
              title={label}
              icon={<DotsThreeIcon size={18} weight="bold" />}
              className="pointer-coarse:size-11"
            />
          }
        />
        <DropdownMenu.Content align="end" className="min-w-44">
          <DropdownMenu.Item className="gap-2" onClick={() => setRenaming(true)}>
            <PencilSimpleIcon size={16} aria-hidden="true" />
            {t('rename')}
          </DropdownMenu.Item>
          <DropdownMenu.Item
            className="gap-2"
            onClick={() => {
              plan.edit(removeCategory(node.id));
              onRemoved();
            }}
          >
            <TrashIcon size={16} aria-hidden="true" />
            {t('remove')}
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu>
      <RenameDialog key={node.name} node={node} open={renaming} onOpenChange={setRenaming} />
    </>
  );
}
