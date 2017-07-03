$( document ).ready(function() {

  var r = new Resumable({
    target: '/upload',
    chunkSize: 5 * 1024 * 1024,
    simultaneousUploads: 15,
  });

  if(!r.support) location.href = '/some-old-crappy-uploader';

  var results         = $('#results'),
      uploadFile      = $('#uploadFiles'),
      browseButton    = $('#browseButton'),
      nothingToUpload = $('[data-nothingToUpload]');
      pauseUpload     = $('#pauseUpload');

  r.assignBrowse(browseButton);

  r.on('fileAdded', function (file, event) {
      var template =
        '<div data-uniqueid="' + file.uniqueIdentifier + '">' +
        '<div class="fileName">' + file.fileName + ' (' + file.file.type + ')' + '</div>' +
        '<div class="large-6 right deleteFile">X</div>' +
        'Upload progress: <progress value="0" max="100" id="progress-meter"></progress>' +
        '</div>';

      results.append(template);
  });

  uploadFile.on('click', function () {
      if (results.children().length > 0) {
          pauseUpload.removeClass('hidden');
          r.upload();
      } else {
          nothingToUpload.fadeIn();
          setTimeout(function () {
            nothingToUpload.fadeOut();
          }, 3000);
      }
  });

  pauseUpload.on('click', function () {
      console.log('pausing upload');
      r.pause();
  });

  $(document).on('click', '.deleteFile', function () {
      var self       = $(this);
          parent     = self.parent(),
          identifier = parent.data('uniqueid'),
          file       = r.getFromUniqueIdentifier(identifier);

      r.removeFile(file);
      parent.remove();
  });

  r.on('fileProgress', function (file) {
      var progress = Math.floor(file.progress() * 100);
      $('#progress-meter').val(progress);
  });

  r.on('fileSuccess', function (file, message) {
      $('[data-uniqueId=' + file.uniqueIdentifier + ']').find('.progress').addClass('success');
      pauseUpload.addClass('hidden');
  });

  r.on('uploadStart', function () {
      $('.alert-box').text('Uploading....');
  });

  r.on('complete', function () {
      $('.alert-box').text('Done Uploading');
  });

  r.on('fileError', function(file, message) {
      console.log('server responded with unexpected status: ' + message);
      setTimeout(function () {
        console.log('retrying ....');
        file.retry();
      }, 3000);
  });

});
