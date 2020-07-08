# sendgrid-man
Sendgrid Man a command line tool which allows you to export your dynamic templates (HTML and Plain) from Sendgrid to files. 
Now you can store them in your version control system.

By default it exports only HTML of active version of each template and names it by the template name `templateName.html` (or `templateName.txt` for plain).
To retrieve all vesions specify `--all` flag. It will retrieve all the versions and name the file it as `templateName__versionID.html`.
To export plain versions use a flac `--include_plain`. It will retrieve the plain content and name the file as `templateName__versionID.txt`.
To overvrite existing files use `--overwrite` flag.

## Usage 

    sendgridman --apikey=Your_Sendgrid_Key --basedir=.
    
or

    export SENDGRID_API_KEY=SG.___
    sendgridman --apikey=$SENDGRID_API_KEY --basedir=.
    
or

    SENDGRID_API_KEY=SG.___ sendgridman --apikey=$SENDGRID_API_KEY --basedir=.

For `--apikey` you should use the Sendgrid ApiKey which have an access to read the templates. Sendgrid shows you an ApiKey only once at create time, if you forgot or lost the key you should create a new one. 
By default all the templates are exported to current working dir. To store in an another dir use `--basedir` flag to set the path to it.


