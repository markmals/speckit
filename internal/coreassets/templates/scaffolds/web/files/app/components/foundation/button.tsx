import type { VariantProps } from "cva";
import type { ButtonProps as AriaButtonProps } from "react-aria-components";

import { Button as AriaButton } from "react-aria-components";

import { cva } from "#/styles/cva.ts";

export const buttonVariants = cva({
    base: "inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-50",
    variants: {
        variant: {
            solid: "bg-neutral-900 text-white hover:bg-neutral-700",
            outline: "border border-neutral-300 hover:bg-neutral-100",
        },
        size: {
            sm: "h-8 px-3 text-sm",
            md: "h-10 px-4 text-sm",
        },
    },
    defaultVariants: { variant: "solid", size: "md" },
});

export interface ButtonProps extends AriaButtonProps, VariantProps<typeof buttonVariants> {}

export function Button({ className, size, variant, ...props }: ButtonProps) {
    return (
        <AriaButton
            className={renderProps =>
                buttonVariants({
                    className: typeof className === "function" ? className(renderProps) : className,
                    size,
                    variant,
                })
            }
            {...props}
        />
    );
}
