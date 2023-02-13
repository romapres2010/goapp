for /f %%a in ('dir /b /s "*.sh"') do (
    git update-index --chmod=+x %%a
)