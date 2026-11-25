import React, {useEffect, useState} from 'react';

type PreviewItem = {
  alt: string;
  height: number;
  src?: string;
  svg?: string;
  width: number;
};

type ImagePreview = {
  images: PreviewItem[];
  index: number;
};

const CONTENT_VISUAL_SELECTOR = [
  '.theme-doc-markdown img',
  '.markdown img',
  '.theme-doc-markdown .docusaurus-mermaid-container > svg',
  '.markdown .docusaurus-mermaid-container > svg',
].join(', ');
const MIN_SCALE = 1;
const MAX_SCALE = 4;
const SCALE_STEP = 0.25;

function uiText() {
  const chinese = document.documentElement.lang.toLowerCase().startsWith('zh');

  if (chinese) {
    return {
      close: '关闭图片预览',
      dialog: '图片预览',
      next: '下一张',
      previous: '上一张',
      zoomIn: '放大',
      zoomOut: '缩小',
    };
  }

  return {
    close: 'Close image preview',
    dialog: 'Image preview',
    next: 'Next image',
    previous: 'Previous image',
    zoomIn: 'Zoom in',
    zoomOut: 'Zoom out',
  };
}

function isContentImage(image: HTMLImageElement): boolean {
  if (image.dataset.noLightbox === 'true') {
    return false;
  }

  if (image.closest('a[href], button, .navbar, .footer, .image-preview-backdrop')) {
    return false;
  }

  return Boolean(image.currentSrc || image.src);
}

function contentVisuals(): HTMLElement[] {
  return Array.from(document.querySelectorAll<HTMLElement>(CONTENT_VISUAL_SELECTOR)).filter((visual) => {
    if (visual instanceof HTMLImageElement) {
      return isContentImage(visual);
    }

    return visual instanceof SVGSVGElement && !visual.closest('.image-preview-backdrop');
  });
}

function clickedVisual(target: EventTarget | null): HTMLElement | null {
  if (!(target instanceof Element) || target.closest('.image-preview-backdrop, .navbar, .footer')) {
    return null;
  }

  const image = target.closest('img');

  if (image instanceof HTMLImageElement && isContentImage(image)) {
    return image;
  }

  const diagram = target.closest('.docusaurus-mermaid-container');
  const svg = diagram?.querySelector(':scope > svg');

  if (
    svg instanceof SVGSVGElement &&
    diagram?.closest('.markdown, .theme-doc-markdown') &&
    !target.closest('a[href]')
  ) {
    return svg;
  }

  return null;
}

function cloneMermaidSvg(svg: SVGSVGElement): string {
  const clone = svg.cloneNode(true) as SVGSVGElement;
  const suffix = `-lightbox-${Math.random().toString(36).slice(2, 8)}`;
  const idNodes = [clone, ...Array.from(clone.querySelectorAll<Element>('[id]'))];
  const ids = [...new Set(idNodes.map((node) => node.id).filter(Boolean))].sort(
    (left, right) => right.length - left.length,
  );
  const renamed = new Map(ids.map((id) => [id, `${id}${suffix}`]));

  const replaceRefs = (value: string) => {
    let next = value;

    for (const [id, nextId] of renamed) {
      next = next.split(`#${id}`).join(`#${nextId}`);
    }

    return next;
  };

  for (const node of [clone, ...Array.from(clone.querySelectorAll<Element>('*'))]) {
    if (node.tagName.toLowerCase() === 'style') {
      node.textContent = replaceRefs(node.textContent ?? '');
    }

    for (const attr of Array.from(node.attributes)) {
      if (attr.name === 'id' || !attr.value.includes('#')) {
        continue;
      }

      const next = replaceRefs(attr.value);

      if (next !== attr.value) {
        node.setAttribute(attr.name, next);
      }
    }
  }

  for (const node of idNodes) {
    const nextId = renamed.get(node.id);

    if (nextId) {
      node.id = nextId;
    }
  }

  clone.removeAttribute('width');
  clone.removeAttribute('height');
  clone.style.maxWidth = 'none';
  clone.style.width = '100%';
  clone.style.height = '100%';

  return clone.outerHTML;
}

function toPreviewItem(visual: HTMLElement): PreviewItem {
  if (visual instanceof HTMLImageElement) {
    return {
      alt: visual.alt,
      height: 0,
      src: visual.currentSrc || visual.src,
      width: 0,
    };
  }

  const svg = visual as SVGSVGElement;
  const box = svg.viewBox.baseVal;
  const rect = svg.getBoundingClientRect();

  return {
    alt: svg.getAttribute('aria-label') || svg.getAttribute('aria-roledescription') || 'Diagram',
    height: box.height > 0 ? box.height : rect.height,
    svg: cloneMermaidSvg(svg),
    width: box.width > 0 ? box.width : rect.width,
  };
}

function clampScale(scale: number): number {
  return Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale));
}

function fittedSize(naturalWidth: number, naturalHeight: number, scale: number) {
  const maxWidth = Math.max(window.innerWidth - 160, 240);
  const maxHeight = Math.max(window.innerHeight - 180, 180);
  const fit = Math.min(1, maxWidth / naturalWidth, maxHeight / naturalHeight);

  return {
    height: Math.round(naturalHeight * fit * scale),
    width: Math.round(naturalWidth * fit * scale),
  };
}

export default function Root({children}: {children: React.ReactNode}) {
  const [preview, setPreview] = useState<ImagePreview | null>(null);
  const [scale, setScale] = useState(MIN_SCALE);
  const [naturalSize, setNaturalSize] = useState<{height: number; width: number} | null>(null);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      const target = event.target;

      const visual = clickedVisual(target);

      if (!visual) {
        return;
      }

      const elements = contentVisuals();
      const index = elements.indexOf(visual);

      if (index < 0) {
        return;
      }

      const images = elements.map(toPreviewItem);

      event.preventDefault();
      setScale(MIN_SCALE);
      setNaturalSize(null);
      setPreview({images, index});
    };

    document.addEventListener('click', handleClick);

    return () => {
      document.removeEventListener('click', handleClick);
    };
  }, []);

  useEffect(() => {
    if (!preview) {
      return undefined;
    }

    const text = uiText();
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setPreview(null);
        return;
      }

      if (event.key === 'ArrowLeft' && preview.images.length > 1) {
        event.preventDefault();
        setScale(MIN_SCALE);
        setNaturalSize(null);
        setPreview((current) =>
          current
            ? {
                ...current,
                index: (current.index - 1 + current.images.length) % current.images.length,
              }
            : current,
        );
        return;
      }

      if (event.key === 'ArrowRight' && preview.images.length > 1) {
        event.preventDefault();
        setScale(MIN_SCALE);
        setNaturalSize(null);
        setPreview((current) =>
          current
            ? {
                ...current,
                index: (current.index + 1) % current.images.length,
              }
            : current,
        );
        return;
      }

      if (event.key === '+' || event.key === '=') {
        event.preventDefault();
        setScale((current) => clampScale(current + SCALE_STEP));
        return;
      }

      if (event.key === '-' || event.key === '_') {
        event.preventDefault();
        setScale((current) => clampScale(current - SCALE_STEP));
      }
    };

    document.body.classList.add('image-preview-open');
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.classList.remove('image-preview-open');
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [preview]);

  const currentImage = preview?.images[preview.index];
  const text = typeof document === 'undefined' ? null : preview ? uiText() : null;
  const displaySize =
    currentImage && naturalSize
      ? fittedSize(naturalSize.width, naturalSize.height, scale)
      : undefined;

  const showImage = (offset: number) => {
    setScale(MIN_SCALE);
    setNaturalSize(null);
    setPreview((current) =>
      current
        ? {
            ...current,
            index: (current.index + offset + current.images.length) % current.images.length,
          }
        : current,
    );
  };

  return (
    <>
      {children}
      {preview && currentImage && text && (
        <div
          aria-label={text.dialog}
          aria-modal="true"
          className="image-preview-backdrop"
          onClick={() => setPreview(null)}
          role="dialog"
        >
          <button
            aria-label={text.close}
            className="image-preview-close"
            onClick={(event) => {
              event.stopPropagation();
              setPreview(null);
            }}
            type="button"
          >
            ×
          </button>
          {preview.images.length > 1 && (
            <button
              aria-label={text.previous}
              className="image-preview-nav image-preview-nav--prev"
              onClick={(event) => {
                event.stopPropagation();
                showImage(-1);
              }}
              type="button"
            >
              ‹
            </button>
          )}
          <div
            className="image-preview-stage"
            onClick={(event) => event.stopPropagation()}
            onWheel={(event) => {
              setScale((current) =>
                clampScale(current + (event.deltaY < 0 ? SCALE_STEP : -SCALE_STEP)),
              );
            }}
          >
            {currentImage.svg ? (
              <div
                aria-label={currentImage.alt}
                className="image-preview-svg"
                dangerouslySetInnerHTML={{__html: currentImage.svg}}
                role="img"
                style={
                  currentImage.width > 0 && currentImage.height > 0
                    ? fittedSize(currentImage.width, currentImage.height, scale)
                    : undefined
                }
              />
            ) : (
              <img
                alt={currentImage.alt}
                className="image-preview-content"
                draggable={false}
                onLoad={(event) => {
                  setNaturalSize({
                    height: event.currentTarget.naturalHeight,
                    width: event.currentTarget.naturalWidth,
                  });
                }}
                src={currentImage.src}
                style={displaySize}
              />
            )}
          </div>
          {preview.images.length > 1 && (
            <button
              aria-label={text.next}
              className="image-preview-nav image-preview-nav--next"
              onClick={(event) => {
                event.stopPropagation();
                showImage(1);
              }}
              type="button"
            >
              ›
            </button>
          )}
          <div className="image-preview-zoom" onClick={(event) => event.stopPropagation()}>
            <button
              aria-label={text.zoomOut}
              disabled={scale <= MIN_SCALE}
              onClick={() => setScale((current) => clampScale(current - SCALE_STEP))}
              type="button"
            >
              −
            </button>
            <span>{Math.round(scale * 100)}%</span>
            <button
              aria-label={text.zoomIn}
              disabled={scale >= MAX_SCALE}
              onClick={() => setScale((current) => clampScale(current + SCALE_STEP))}
              type="button"
            >
              +
            </button>
          </div>
        </div>
      )}
    </>
  );
}
