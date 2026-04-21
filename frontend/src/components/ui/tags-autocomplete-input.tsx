'use client';

import { forwardRef, useEffect, useMemo, useRef, useState, type ChangeEvent, type KeyboardEvent } from 'react';
import { Check, X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { TruncatedText } from '@/components/truncated-text';
import { Popover, PopoverContent, PopoverTrigger } from './popover';

const MAX_DISPLAY = 100;

interface TagsAutocompleteInputProps {
  value: string[];
  onChange: (tags: string[]) => void;
  placeholder?: string;
  className?: string;
  suggestions?: string[];
  isLoading?: boolean;
}

export const TagsAutocompleteInput = forwardRef<HTMLDivElement, TagsAutocompleteInputProps>(
  ({ value = [], onChange, placeholder, className, suggestions = [], isLoading }, _ref) => {
    const [inputValue, setInputValue] = useState('');
    const [open, setOpen] = useState(false);
    const [isComposing, setIsComposing] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    const filteredSuggestions = useMemo(() => {
      const result: string[] = [];
      const q = inputValue.trim().toLowerCase();
      for (const suggestion of suggestions) {
        if (value.includes(suggestion)) continue;
        if (q && !suggestion.toLowerCase().includes(q)) continue;
        result.push(suggestion);
        if (result.length >= MAX_DISPLAY) break;
      }
      return result;
    }, [inputValue, suggestions, value]);

    const handleInputChange = (event: ChangeEvent<HTMLInputElement>) => {
      setInputValue(event.target.value);
      if (event.target.value && !open && suggestions.length > 0) {
        setOpen(true);
      } else if (!event.target.value) {
        setOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
      if (isComposing) return;

      if (event.key === 'Enter' || event.key === ',') {
        event.preventDefault();
        const newTag = inputValue.trim();
        if (newTag && !value.includes(newTag)) {
          onChange([...value, newTag]);
        }
        setInputValue('');
        setOpen(false);
      } else if (event.key === 'Backspace' && !inputValue && value.length > 0) {
        onChange(value.slice(0, -1));
      } else if (event.key === 'Escape') {
        setOpen(false);
      }
    };

    const removeTag = (tagToRemove: string) => {
      onChange(value.filter((tag) => tag !== tagToRemove));
    };

    const handleSelectSuggestion = (suggestion: string) => {
      if (!value.includes(suggestion)) {
        onChange([...value, suggestion]);
      }
      setInputValue('');
      setOpen(false);
      inputRef.current?.focus();
    };

    const handleInputBlur = () => {
      if (isComposing) return;
      const newTag = inputValue.trim();
      if (newTag && !value.includes(newTag)) {
        onChange([...value, newTag]);
      }
      setInputValue('');
      setOpen(false);
    };

    const handleContainerClick = () => {
      inputRef.current?.focus();
      if (suggestions.length > 0 && inputValue) {
        setOpen(true);
      }
    };

    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        const target = event.target as HTMLElement;
        if (!target.closest('[data-tags-input-container]')) {
          setOpen(false);
        }
      };

      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    return (
      <div
        ref={containerRef}
        data-tags-input-container
        className={cn(
          'border-input bg-background ring-offset-background focus-within:ring-ring flex min-h-10 w-full flex-wrap gap-1 rounded-md border px-3 py-2 text-sm focus-within:ring-2 focus-within:ring-offset-2',
          className
        )}
        onClick={handleContainerClick}
      >
        {value.map((tag) => (
          <div key={tag} className='bg-secondary text-secondary-foreground flex items-center gap-1 rounded-sm px-2 py-0.5'>
            <span className='text-xs'>{tag}</span>
            <button
              type='button'
              onClick={() => removeTag(tag)}
              className='text-secondary-foreground/80 hover:text-secondary-foreground focus:outline-none'
              aria-label={`Remove ${tag} tag`}
            >
              <X className='h-3 w-3' />
            </button>
          </div>
        ))}
        <Popover open={open} onOpenChange={setOpen} modal={false}>
          <PopoverTrigger asChild>
            <input
              ref={inputRef}
              type='text'
              value={inputValue}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              onBlur={handleInputBlur}
              onFocus={() => {
                if (suggestions.length > 0 && inputValue) {
                  setOpen(true);
                }
              }}
              onCompositionStart={() => setIsComposing(true)}
              onCompositionEnd={() => setIsComposing(false)}
              placeholder={value.length === 0 ? placeholder : ''}
              className='placeholder:text-muted-foreground min-w-[80px] flex-1 bg-transparent outline-none'
            />
          </PopoverTrigger>
          {(isLoading || filteredSuggestions.length > 0 || inputValue.trim()) && (
            <PopoverContent
              className='w-[var(--radix-popover-trigger-width)] max-w-[var(--radix-popover-trigger-width)] p-0'
              align='start'
              onOpenAutoFocus={(event) => event.preventDefault()}
              container={containerRef.current ?? undefined}
            >
              <div className='max-h-[200px] overflow-y-auto p-1'>
                {isLoading ? (
                  <div className='text-muted-foreground p-2 text-sm'>Loading...</div>
                ) : filteredSuggestions.length > 0 ? (
                  filteredSuggestions.map((suggestion) => (
                    <div
                      key={suggestion}
                      className='hover:bg-accent flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm'
                      onMouseDown={(event) => {
                        event.preventDefault();
                        handleSelectSuggestion(suggestion);
                      }}
                    >
                      <Check className='text-muted-foreground h-4 w-4 shrink-0' />
                      <TruncatedText className='flex-1'>{suggestion}</TruncatedText>
                    </div>
                  ))
                ) : inputValue.trim() ? (
                  <div className='text-muted-foreground p-2 text-sm'>Press Enter to add &quot;{inputValue}&quot;</div>
                ) : null}
              </div>
            </PopoverContent>
          )}
        </Popover>
      </div>
    );
  }
);

TagsAutocompleteInput.displayName = 'TagsAutocompleteInput';
