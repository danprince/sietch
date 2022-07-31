import { useMemo, useEffect } from "preact/hooks";

const URL_REGEX = /(?:https?:\/\/)?(?:[^\s.]+\.)+[^A-Z\s.][/\w?&=]+/;
const EMAIL_REGEX = /\S+@\S+\.\S/g;
const LINK_REGEX = new RegExp(`(${URL_REGEX.source}|${EMAIL_REGEX.source})`);

interface LinkedProps {
  /**
   * Linked's children must be a single string, so that we can run the splitting
   * regex on it. Doesn't work with nested elements.
   */
  children: string;
}

const Linked = ({ children }: LinkedProps) => {
  // Prevent selecting containing object when a link is clicked.
  const onClick = e => e.stopPropagation();

  const newChildren = useMemo(() => {
    let parts = children.split(LINK_REGEX);

    return parts.map((part) => {
      if (LINK_REGEX.test(part)) {
        let linkProps = {
          target: "_blank",
          rel: "noopener noreferrer",
          className: "underline",
          title: "External Link",
          href: part,
          onClick,
        };

        if (!linkProps.href.startsWith("http")) {
          linkProps.href = `http://${linkProps.href}`;
        }

        if (part.includes("@")) {
          linkProps.href = `mailto:${part}`;
        }

        return <a {...linkProps}>{part}</a>;
      } else {
        return part;
      }
    });
  }, [children]);

  useEffect(() => {
      throw new Error("ffs")
  }, []);

  return (
    // Hack to prevent us needing to provide keys for the children
    <>{newChildren}</>
  );
};

export default Linked;
